# -*- coding: utf-8 -*-
"""
Star-Fire smart 负载均衡算法 —— 新两阶段算法仿真测试（含带宽维度）
================================================================
忠实复刻修改后的 internal/models/server.go 逻辑。

新算法（两阶段）：
  阶段1（性能评分）：对所有合格 client 计算性能综合评分
    - 维度：容量/延迟/失败率/稳定性/产能/上行带宽（不含会员等级）
    - 权重：cap=0.20, lat=0.20, fail=0.15, stab=0.10, serv=0.10, bw=0.25
    - 选出性能评分最高的前 N 名候选（N = LBCandidateCount，默认=重试次数 3）
  阶段2（会员权重）：在候选中按会员等级权重加权随机选择
    - normal=1.0, vip=3.0, svip=8.0（体现充值价值：svip > vip > normal）

过滤阶段（Predicate）：
  - clientHealthy: 在线 && 有控制连接 && 延迟 < 阈值 && 注册了模型
  - priceEligible: 价格 <= 上限
  - connectionLimitEligible: 可用连接数 > 0（activeConnections < maxConnections）
    - 连接池：normal=1, vip=5, svip=50（饱和时被过滤）

测试场景：1 小时内 10000 个请求，500 个客户端（参数平均分布，300 个不断注册/退出）。
"""

import random
import statistics
from collections import defaultdict

# ============ 常量（与 Go 源码一致） ============
MAXLATENCE = 30000
MAX_LATENCY_MS = 30000
LB_TARGET_ONLINE_SEC = 3600
LB_TARGET_TOKENS_PER_HOUR = 100000

# 性能权重（阶段1）
W_CAP, W_LAT, W_FAIL, W_STAB, W_SERV, W_BW = 0.20, 0.20, 0.15, 0.10, 0.10, 0.25
JITTER = 0.05
CANDIDATE_COUNT = 3  # 默认 = 重试次数

# 会员连接池上限
MAX_CONN = {"normal": 1, "vip": 5, "svip": 50}
# 会员权重（阶段2），体现充值价值
MEM_WEIGHT = {"normal": 1.0, "vip": 3.0, "svip": 8.0}

# 带宽
CLIENT_BANDWIDTH_MBPS = 10   # 默认上行带宽
BANDWIDTH_TARGET_MBPS = 50   # 带宽评分目标


def effective_max_conn(c):
    """有效连接数上限：优先使用 client 自定义上限（Python 滑块，0~会员上限），否则用会员默认。"""
    base = MAX_CONN[c["membership"]]
    override = c.get("max_conn_override", 0)
    if override > 0 and override <= base:
        return override
    return base


# ============ 评分函数（与 Go 源码逐行对应） ============
def capacity_score(max_conn, active_conn):
    if max_conn <= 0:
        return 0.0
    avail = max_conn - active_conn
    if avail < 0:
        avail = 0
    return avail / max_conn


def latency_score(latency):
    if latency <= 0:
        return 1.0
    score = 1 - latency / MAXLATENCE
    return max(0.0, min(1.0, score))


def failure_score(failures):
    return 1.0 / (1.0 + failures)


def stability_score(total_online_sec, disconnect_count):
    avg_online = total_online_sec / (disconnect_count + 1)
    score = avg_online / LB_TARGET_ONLINE_SEC
    return max(0.0, min(1.0, score))


def service_score(total_tokens, total_online_sec):
    if total_online_sec <= 0:
        return 0
    tokens_per_hour = total_tokens / (total_online_sec / 3600.0)
    score = tokens_per_hour / LB_TARGET_TOKENS_PER_HOUR
    return max(0.0, min(1.0, score))


def bandwidth_score(bw):
    if bw <= 0:
        return 0
    score = bw / BANDWIDTH_TARGET_MBPS
    return max(0.0, min(1.0, score))


def perf_score(c):
    """阶段1：性能综合评分（不含会员等级）"""
    max_conn = effective_max_conn(c)
    bw = c["bandwidth"] if c["bandwidth"] > 0 else CLIENT_BANDWIDTH_MBPS
    return (W_CAP * capacity_score(max_conn, c["active_conn"]) +
            W_LAT * latency_score(c["latency"]) +
            W_FAIL * failure_score(c["failures"]) +
            W_STAB * stability_score(c["total_online_sec"], c["disconnect_count"]) +
            W_SERV * service_score(c["total_tokens"], c["total_online_sec"]) +
            W_BW * bandwidth_score(bw))


def client_healthy(c, model):
    if model not in c["models"]:
        return False
    return c["online"] and c["control_conn"] and c["latency"] < MAX_LATENCY_MS


def connection_limit_eligible(c):
    """可用连接数必须 > 0，否则过滤"""
    limit = effective_max_conn(c)
    if limit <= 0:
        return False
    return c["active_conn"] < limit


def pick_smart(eligible):
    """两阶段：性能评分选前 N 候选，再按会员权重加权随机选择"""
    if not eligible:
        return None

    # 阶段1：性能评分 + 扰动，降序排序
    scored = [(c, perf_score(c) * (1 + (random.random() * 2 - 1) * JITTER)) for c in eligible]
    scored.sort(key=lambda x: x[1], reverse=True)

    # 取前 N 候选
    n = min(CANDIDATE_COUNT, len(scored))
    candidates = scored[:n]

    # 阶段2：按会员权重加权随机
    total_weight = sum(MEM_WEIGHT[c["membership"]] for c, _ in candidates)
    r = random.random() * total_weight
    for c, s in candidates:
        r -= MEM_WEIGHT[c["membership"]]
        if r <= 0:
            return c
    return candidates[-1][0]


def load_balance(model, clients, exclude=None):
    exclude = exclude or set()
    eligible = []
    for cid, c in clients.items():
        if cid in exclude:
            continue
        if not client_healthy(c, model):
            continue
        if not connection_limit_eligible(c):
            continue
        eligible.append(c)
    if not eligible:
        return None, []
    return pick_smart(eligible), eligible


# ============ 数据生成 ============
def gen_clients(n, seed):
    rng = random.Random(seed)
    clients = {}
    memberships = ["normal", "vip", "svip"]
    for i in range(n):
        cid = f"c{i:04d}"
        mem = rng.choice(memberships)
        clients[cid] = {
            "id": cid,
            "membership": mem,
            "models": {"gpt-4o"},
            "online": True,
            "control_conn": True,
            "latency": rng.randint(20, 500),
            "active_conn": 0,
            "failures": rng.randint(0, 5),
            "total_online_sec": rng.randint(600, 7200),
            "disconnect_count": rng.randint(0, 10),
            "total_tokens": rng.randint(0, 200000),
            "bandwidth": rng.uniform(5, 100),  # 5~100 Mbps 平均分布
            "max_conn_override": 0,  # 0 = 使用会员默认；部分 client 会设置更低的自定义上限
            "churn": False,
        }
    return clients


# ============ 主测试 ============
def run_test():
    random.seed(42)
    N_CLIENTS = 500
    N_CHURN = 300
    N_REQUESTS = 10000
    MODEL = "gpt-4o"

    clients = gen_clients(N_CLIENTS, seed=1)
    churn_ids = list(clients.keys())[:N_CHURN]
    for cid in churn_ids:
        clients[cid]["churn"] = True

    # 模拟部分用户用滑块设置了更低的自定义连接数上限（0 = 自动/会员默认）
    rng = random.Random(7)
    for c in clients.values():
        base = MAX_CONN[c["membership"]]
        if rng.random() < 0.3:  # 30% 的 client 设置了自定义上限
            c["max_conn_override"] = rng.randint(1, base)

    picks = defaultdict(int)
    picks_by_id = defaultdict(int)
    eligible_counts = []
    svip_saturated_picks = 0
    vip_normal_pick = 0
    total_picks = 0
    no_eligible = 0

    for req in range(N_REQUESTS):
        # churn：300 个 client 每 10 个请求切换一次在线状态
        if req % 10 == 0:
            cid = churn_ids[(req // 10) % N_CHURN]
            c = clients[cid]
            c["online"] = not c["online"]
            c["control_conn"] = c["online"]
            if c["online"]:
                c["total_online_sec"] += 60

        picked, eligible = load_balance(MODEL, clients)
        if picked is None:
            no_eligible += 1
            continue

        total_picks += 1
        mem = picked["membership"]
        picks[mem] += 1
        picks_by_id[picked["id"]] += 1
        eligible_counts.append(len(eligible))

        if mem == "svip" and picked["active_conn"] >= 50:
            svip_saturated_picks += 1
        if mem in ("vip", "normal"):
            vip_normal_pick += 1

        # 请求被分配给 picked，增加其 active_conn；随机释放
        picked["active_conn"] += 1
        for c in clients.values():
            if c["active_conn"] > 0 and random.random() < 0.3:
                c["active_conn"] -= 1

    # ============ 结果输出 ============
    print("=" * 72)
    print("Star-Fire smart 负载均衡 —— 新两阶段算法（含带宽）仿真结果")
    print("=" * 72)
    print(f"客户端总数: {N_CLIENTS}  (churn: {N_CHURN} 个不断注册/退出)")
    print(f"请求总数: {N_REQUESTS}  (1 小时)")
    print(f"成功分配: {total_picks}  无可用client: {no_eligible}")
    print(f"平均 eligible 数: {statistics.mean(eligible_counts):.1f}")
    print(f"连接池: normal={MAX_CONN['normal']}, vip={MAX_CONN['vip']}, svip={MAX_CONN['svip']}")
    print(f"会员权重: normal={MEM_WEIGHT['normal']}, vip={MEM_WEIGHT['vip']}, svip={MEM_WEIGHT['svip']}")
    print()

    print("--- 各会员被 pick 次数 ---")
    for mem in ["normal", "vip", "svip"]:
        cnt = picks[mem]
        pct = cnt / total_picks * 100 if total_picks else 0
        print(f"  {mem:8s}: {cnt:6d} 次  ({pct:5.1f}%)")
    print()

    print("--- 关键问题验证 ---")
    print(f"  SVIP 已饱和(active>=50)仍被 pick 次数: {svip_saturated_picks}")
    print(f"  VIP/normal 被 pick 次数: {vip_normal_pick}  "
          f"({vip_normal_pick/total_picks*100:.1f}%)")
    print()

    picked_ids_by_mem = defaultdict(set)
    for cid, cnt in picks_by_id.items():
        picked_ids_by_mem[clients[cid]["membership"]].add(cid)
    print("--- 被 pick 到的不同 client 数（覆盖度） ---")
    for mem in ["normal", "vip", "svip"]:
        total_mem = sum(1 for c in clients.values() if c["membership"] == mem)
        print(f"  {mem:8s}: 被pick {len(picked_ids_by_mem[mem]):4d} / 总 {total_mem:4d}  "
              f"({len(picked_ids_by_mem[mem])/max(total_mem,1)*100:.1f}%)")
    print()

    print("--- 各会员平均并发连接数（饱和度） ---")
    for mem in ["normal", "vip", "svip"]:
        mem_clients = [c for c in clients.values() if c["membership"] == mem]
        avg = statistics.mean(c["active_conn"] for c in mem_clients)
        print(f"  {mem:8s}: 平均 active_conn = {avg:.2f}  (上限 {MAX_CONN[mem]})")
    print()


if __name__ == "__main__":
    run_test()
