"""
生产级负载均衡并发模拟（不启动服务，纯内存，离散事件模拟 DES）。

场景：
- 10,000 个消费者，每个 10~100 个 agent 请求（总请求量 10 万 ~ 100 万）。
- 50 个 Direct 后端（平台自建，MaxConns=80）。
- 5,000 个个人贡献者节点（personal client），提供模型服务，normal 默认 1 并发。
- 节点不断上下线（churn）。
- 统计：请求失败率、Direct/贡献者分配、超配、计费、冷却、上下线影响、prefix 命中率。

算法忠实还原 internal/models/server.go 的生产逻辑：
- stability（默认）：PickDirect 优先 → 失败回退 LoadBalanceWithTolerance → pickSmart 两阶段。
- cost：PickCheapest 合并两池按单价分层，层内可靠性竞争。
- balanced：LoadBalanceBalanced（perfScore>=0.5 过滤）→ 溢出 Direct。
- 连接上限：normal=1, vip=5, svip=50；Direct MaxConns=80。
- 冷却：失败指数退避（base 5s × 2^(n-1)，上限 5min）。
- 推理耗时 + 超时 + 重试（MAX_CHAT_RETRY=3，总超时 60s）。
- 计费：IPPM×0.7 + OPPM×0.3 单价；按实际服务 client 价格计费。

运行：python docs/lb_production_sim.py
可用环境变量覆盖规模：SIM_CONSUMERS / SIM_DIRECT / SIM_CONTRIBUTORS / SIM_DURATION
"""
import heapq
import math
import os
import random
import time
from dataclasses import dataclass

# ============ 常量（与 Go 生产一致） ============
MAXLATENCE = 30000          # public.MAXLATENCE（ms）
MAX_CHAT_RETRY = 3          # public.MAX_CHAT_RETRY
CHAT_RETRY_TOTAL_TIMEOUT = 60  # 秒
LB_TARGET_ONLINE_SEC = 3600
LB_TARGET_TOKENS_PER_HOUR = 100000
LB_EMA_ALPHA = 0.3
LB_JITTER = 0.05
LB_CANDIDATE_COUNT = 3
LB_BALANCED_MIN_SCORE = 0.5
LB_COOLDOWN_BASE_MS = 5000
LB_COOLDOWN_MAX_MS = 300000
LB_HRW_SUBSET_SIZE = 0      # 0 = 关闭 HRW 子集

# smart 权重（默认兜底值）
W_CAP, W_LAT, W_FAIL, W_STAB, W_SERV, W_BW = 0.20, 0.20, 0.15, 0.10, 0.10, 0.25
# 会员权重
W_NORMAL, W_VIP, W_SVIP = 1.0, 3.0, 8.0
# 会员连接上限
MAX_CONN = {'normal': 1, 'vip': 5, 'svip': 50}

# ============ 拓扑规模（可用环境变量覆盖，便于快速验证） ============
def _env_int(name, default):
    try:
        return int(os.environ.get(name, default))
    except (TypeError, ValueError):
        return default


N_CONSUMERS = _env_int("SIM_CONSUMERS", 10000)
REQ_PER_CONSUMER_MIN, REQ_PER_CONSUMER_MAX = 10, 100
N_DIRECT = _env_int("SIM_DIRECT", 50)
N_CONTRIBUTORS = _env_int("SIM_CONTRIBUTORS", 5000)
DIRECT_MAX_CONNS = 80
SIM_DURATION = _env_int("SIM_DURATION", 1800)   # 模拟时长（秒）

# 模型
MODELS = [
    # name, direct_price(ippm, oppm), contributor_price(ippm, oppm)
    ("qwen3-32b", (1.0, 2.0), (0.8, 1.6)),
    ("qwen3-72b", (2.0, 4.0), (1.6, 3.2)),
    ("glm-5",     (1.5, 3.0), (1.2, 2.4)),
    ("deepseek-v4",(3.0, 6.0), (2.4, 4.8)),
]
MODEL_NAMES = [m[0] for m in MODELS]

# 上下线（churn）参数
CHURN_INTERVAL_SEC = 5.0     # 每 5s 触发一次上下线
CHURN_FRACTION = 0.02        # 每次上下线 2% 的贡献者节点
DIRECT_CHURN_FRACTION = 0.01 # Direct 每次 1%

# ============ 全局模拟时钟（DES） ============
SIM_NOW = 0.0


# ============ 评分函数（与 Go 一致） ============
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
    s = 1 - latency / MAXLATENCE
    return max(0.0, min(1.0, s))


def failure_score(failures):
    return 1.0 / (1.0 + failures)


def stability_score(total_online_sec, disconnect_count):
    avg = total_online_sec / (disconnect_count + 1)
    s = avg / LB_TARGET_ONLINE_SEC
    return max(0.0, min(1.0, s))


def service_score(total_tokens, total_online_sec):
    if total_online_sec <= 0:
        return 0.0
    tph = total_tokens / (total_online_sec / 3600.0)
    s = tph / LB_TARGET_TOKENS_PER_HOUR
    return max(0.0, min(1.0, s))


def bandwidth_score(bw):
    if bw <= 0:
        return 0.0
    target = 50.0
    s = bw / target
    return max(0.0, min(1.0, s))


def cost_price(ippm, oppm):
    return ippm * 0.7 + oppm * 0.3


# ============ 节点 ============
@dataclass
class Contributor:
    cid: str
    model: str
    membership: str
    ippm: float
    oppm: float
    latency: int = 200
    latency_ema: float = 200.0
    active: int = 0
    failures: int = 0
    cooldown_until: float = 0.0
    online: bool = True
    total_online_sec: int = 3600
    disconnect_count: int = 0
    total_tokens: int = 50000
    bw: float = 100.0
    cache_hit_ema: float = 0.0
    # 统计
    served: int = 0
    failed: int = 0

    @property
    def max_conn(self):
        return MAX_CONN[self.membership]

    def get_latency(self):
        return int(self.latency_ema)

    def cached_scores(self):
        # 模拟 ScoreRefresher 已刷新
        return (stability_score(self.total_online_sec, self.disconnect_count),
                service_score(self.total_tokens, self.total_online_sec))

    def in_cooldown(self):
        return self.cooldown_until > SIM_NOW

    def trip_cooldown(self):
        n = max(1, self.failures)
        d = LB_COOLDOWN_BASE_MS << (n - 1)
        if n > 30 or d > LB_COOLDOWN_MAX_MS or d <= 0:
            d = LB_COOLDOWN_MAX_MS
        self.cooldown_until = SIM_NOW + d / 1000.0

    def perf_score(self):
        w = (W_CAP, W_LAT, W_FAIL, W_STAB, W_SERV, W_BW)
        stab, serv = self.cached_scores()
        return (w[0] * capacity_score(self.max_conn, self.active) +
                w[1] * latency_score(self.get_latency()) +
                w[2] * failure_score(self.failures) +
                w[3] * stab +
                w[4] * serv +
                w[5] * bandwidth_score(self.bw))


@dataclass
class DirectBackend:
    bid: str
    model: str
    ippm: float
    oppm: float
    priority: int = 0
    max_conns: int = DIRECT_MAX_CONNS
    active: int = 0
    failures: int = 0
    cooldown_until: float = 0.0
    healthy: bool = True
    latency_ema: float = 150.0
    reliability_ema: float = 0.5
    # 统计
    served: int = 0
    failed: int = 0

    def get_active(self):
        return self.active

    def in_cooldown(self):
        return self.cooldown_until > SIM_NOW

    def trip_cooldown(self):
        n = max(1, self.failures)
        d = LB_COOLDOWN_BASE_MS << (n - 1)
        if n > 30 or d > LB_COOLDOWN_MAX_MS or d <= 0:
            d = LB_COOLDOWN_MAX_MS
        self.cooldown_until = SIM_NOW + d / 1000.0

    def is_healthy(self):
        return self.healthy

    def quality_score(self):
        return (W_CAP * capacity_score(self.max_conns, self.active) +
                W_LAT * latency_score(int(self.latency_ema)) +
                W_FAIL * failure_score(self.failures) +
                W_STAB * self.reliability_ema +
                W_SERV * 1.0 +
                W_BW * 1.0)


# ============ 拓扑 ============
class Topology:
    def __init__(self, rng):
        self.rng = rng
        self.contributors = {}   # cid -> Contributor
        self.direct = {}         # bid -> DirectBackend
        self.contributors_by_model = {m: [] for m in MODEL_NAMES}
        self.direct_by_model = {m: [] for m in MODEL_NAMES}
        self._init_direct()
        self._init_contributors()

    def _init_direct(self):
        for i in range(N_DIRECT):
            model = MODEL_NAMES[i % len(MODEL_NAMES)]
            ippm, oppm = MODELS[MODEL_NAMES.index(model)][1]
            b = DirectBackend(
                bid=f"d{i}", model=model, ippm=ippm, oppm=oppm,
                priority=self.rng.randint(0, 2),
                latency_ema=self.rng.uniform(80, 300),
                reliability_ema=self.rng.uniform(0.6, 0.99),
            )
            self.direct[b.bid] = b
            self.direct_by_model[model].append(b)

    def _init_contributors(self):
        for i in range(N_CONTRIBUTORS):
            model = MODEL_NAMES[i % len(MODEL_NAMES)]
            ippm, oppm = MODELS[MODEL_NAMES.index(model)][2]
            m = self.rng.choices(['normal', 'vip', 'svip'], weights=[0.8, 0.15, 0.05])[0]
            c = Contributor(
                cid=f"c{i}", model=model, membership=m, ippm=ippm, oppm=oppm,
                latency=self.rng.randint(50, 2000),
                latency_ema=self.rng.randint(50, 2000),
                total_online_sec=self.rng.randint(600, 7200),
                disconnect_count=self.rng.randint(0, 5),
                total_tokens=self.rng.randint(10000, 200000),
                bw=self.rng.uniform(10, 200),
            )
            self.contributors[c.cid] = c
            self.contributors_by_model[model].append(c)

    def contributors_for(self, model):
        return self.contributors_by_model[model]

    def direct_for(self, model):
        return self.direct_by_model[model]

    def churn(self):
        """模拟节点上下线。返回 (上线数, 下线数)。"""
        up = down = 0
        # 贡献者 churn
        cs = list(self.contributors.values())
        n = max(1, int(len(cs) * CHURN_FRACTION))
        for c in self.rng.sample(cs, min(n, len(cs))):
            if c.online:
                c.online = False
                c.disconnect_count += 1
                down += 1
            else:
                c.online = True
                c.active = 0
                c.failures = 0
                c.cooldown_until = 0.0
                up += 1
        # Direct churn（健康抖动）
        ds = list(self.direct.values())
        nd = max(1, int(len(ds) * DIRECT_CHURN_FRACTION))
        for b in self.rng.sample(ds, min(nd, len(ds))):
            b.healthy = not b.healthy
            if not b.healthy:
                down += 1
            else:
                b.active = 0
                b.failures = 0
                up += 1
        return up, down


# ============ 路由算法（忠实还原 server.go） ============
class Router:
    def __init__(self, topo, rng):
        self.topo = topo
        self.rng = rng

    # ---- eligibleClients（众包） ----
    def eligible_clients(self, model, exclude):
        out = []
        for c in self.topo.contributors_for(model):
            if exclude.get(c.cid):
                continue
            if not c.online:
                continue
            if c.in_cooldown():
                continue
            if c.get_latency() >= MAXLATENCE:
                continue
            if c.active >= c.max_conn:
                continue
            out.append(c)
        return out

    # ---- pickSmart 两阶段 ----
    def pick_smart(self, eligible):
        if not eligible:
            return None
        scored = []
        for c in eligible:
            s = c.perf_score() * (1 + (self.rng.random() * 2 - 1) * LB_JITTER)
            scored.append((s, c))
        scored.sort(key=lambda x: x[0], reverse=True)
        n = min(LB_CANDIDATE_COUNT, len(scored))
        candidates = scored[:n]
        weights = [W_NORMAL if c.membership == 'normal' else (W_VIP if c.membership == 'vip' else W_SVIP)
                   for _, c in candidates]
        total = sum(weights)
        if total <= 0:
            return candidates[0][1]
        r = self.rng.random() * total
        for (_, c), w in zip(candidates, weights):
            r -= w
            if r <= 0:
                return c
        return candidates[-1][1]

    # ---- loadBalanceExcluding（stability/balanced 共用） ----
    def load_balance(self, model, exclude, min_score=0.0):
        eligible = self.eligible_clients(model, exclude)
        if not eligible:
            return None
        if min_score > 0:
            eligible = [c for c in eligible if c.perf_score() >= min_score]
            if not eligible:
                return None
        return self.pick_smart(eligible)

    # ---- PickDirect ----
    def pick_direct(self, model, exclude):
        best = None
        best_prio = float('inf')
        best_load = float('inf')
        for b in self.topo.direct_for(model):
            if exclude.get("direct:" + b.bid):
                continue
            if not b.is_healthy() or b.in_cooldown():
                continue
            maxc = b.max_conns if b.max_conns > 0 else 1
            if b.get_active() >= maxc:
                continue
            load = b.get_active() / maxc
            if b.priority < best_prio or (b.priority == best_prio and load < best_load):
                best, best_prio, best_load = b, b.priority, load
        return best

    # ---- PickCheapest（cost） ----
    def pick_cheapest(self, model, exclude):
        clients = self.eligible_clients(model, exclude)
        backends = []
        for b in self.topo.direct_for(model):
            if exclude.get("direct:" + b.bid):
                continue
            if not b.is_healthy() or b.in_cooldown():
                continue
            maxc = b.max_conns if b.max_conns > 0 else 1
            if b.get_active() >= maxc:
                continue
            backends.append(b)
        cands = []
        for c in clients:
            cands.append((cost_price(c.ippm, c.oppm), c, None))
        for b in backends:
            cands.append((cost_price(b.ippm, b.oppm), None, b))
        if not cands:
            return None, None
        cands.sort(key=lambda x: x[0])
        min_price = cands[0][0]
        layer = [x for x in cands if x[0] - min_price <= 1e-9]
        layer_clients = [x[1] for x in layer if x[1] is not None]
        layer_backends = [x[2] for x in layer if x[2] is not None]
        if not layer_clients:
            best = min(layer_backends, key=lambda b: b.get_active() / (b.max_conns or 1))
            return None, best
        if not layer_backends:
            return self.pick_smart(layer_clients), None
        # 两类都有：可靠性竞争
        best_community = max(layer_clients, key=lambda c: c.perf_score())
        best_direct = min(layer_backends, key=lambda b: b.get_active() / (b.max_conns or 1))
        if best_community.perf_score() >= best_direct.quality_score():
            return best_community, None
        return None, best_direct


# ============ 推理 + 计费 ============
def simulate_inference(rng, node, is_direct):
    """模拟一次推理。返回 (成功, 耗时秒, 输出tokens)。"""
    if is_direct:
        base = rng.uniform(0.5, 3.0)
        fail_p = 0.01
    else:
        base = rng.uniform(1.0, 8.0)
        fail_p = 0.05
    cap = node.max_conns if is_direct else node.max_conn
    load_factor = 1.0 + (node.active / max(1, cap)) * 0.5
    dur = base * load_factor
    out_tokens = rng.randint(200, 2000)
    if rng.random() < fail_p:
        return False, dur, out_tokens
    return True, dur, out_tokens


# ============ Prefix 缓存（统计命中率） ============
class PrefixCache:
    """每个模型一个 prefix 缓存。热门 prefix 复用率高，模拟真实 prefix 命中。"""
    def __init__(self, rng):
        self.rng = rng
        self.hot = {m: [f"pfx-{m}-{i}" for i in range(200)] for m in MODEL_NAMES}
        self.cache = {m: set() for m in MODEL_NAMES}
        self.hits = 0
        self.total = 0

    def check(self, model):
        """返回 (是否命中, prefix)。"""
        self.total += 1
        # 80% 概率用热门 prefix（复用），20% 冷门
        if self.rng.random() < 0.8:
            pfx = self.rng.choice(self.hot[model])
        else:
            pfx = f"pfx-{model}-cold-{self.rng.randint(0, 100000)}"
        if pfx in self.cache[model]:
            self.hits += 1
            return True, pfx
        self.cache[model].add(pfx)
        return False, pfx


# ============ 主模拟（DES） ============
def run_simulation(routing, seed):
    global SIM_NOW
    rng = random.Random(seed)
    topo = Topology(rng)
    router = Router(topo, rng)
    pfx = PrefixCache(rng)

    # 事件队列：(时间, 序号, 类型, 负载)
    events = []
    seq = 0

    # 生成消费者请求到达事件
    total_req = 0
    for i in range(N_CONSUMERS):
        n = rng.randint(REQ_PER_CONSUMER_MIN, REQ_PER_CONSUMER_MAX)
        model = rng.choice(MODEL_NAMES)
        total_req += n
        for _ in range(n):
            t = rng.uniform(0, SIM_DURATION)
            heapq.heappush(events, (t, seq, 'req', (model,)))
            seq += 1

    # 预排 churn 事件
    t = CHURN_INTERVAL_SEC
    while t < SIM_DURATION:
        heapq.heappush(events, (t, seq, 'churn', None))
        seq += 1
        t += CHURN_INTERVAL_SEC

    stats = {
        'direct': 0, 'community': 0, 'unavailable': 0,
        'over_capacity': 0, 'failed': 0, 'succeeded': 0,
        'retries': 0, 'cost': 0.0, 'out_tokens': 0,
        'direct_served': 0, 'community_served': 0,
        'direct_failed': 0, 'community_failed': 0,
        'churn_up': 0, 'churn_down': 0,
        'prefix_hits': 0, 'prefix_total': 0,
    }
    max_over = 0
    start = time.time()

    while events:
        t, _, etype, payload = heapq.heappop(events)
        SIM_NOW = t

        if etype == 'churn':
            up, down = topo.churn()
            stats['churn_up'] += up
            stats['churn_down'] += down
            continue

        if etype == 'done':
            kind, nid = payload
            if kind == 'direct':
                b = topo.direct[nid]
                b.active -= 1
            else:
                c = topo.contributors[nid]
                c.active -= 1
            continue

        # 请求到达
        model = payload[0]
        hit, _ = pfx.check(model)
        if hit:
            stats['prefix_hits'] += 1
        stats['prefix_total'] += 1

        exclude = {}
        success = False
        attempt_start = SIM_NOW
        for attempt in range(MAX_CHAT_RETRY):
            if SIM_NOW - attempt_start > CHAT_RETRY_TOTAL_TIMEOUT:
                break
            # 0. Direct 优先（stability 非 cost）
            if routing != 'cost':
                b = router.pick_direct(model, exclude)
                if b is not None:
                    exclude["direct:" + b.bid] = True
                    b.active += 1
                    if b.active > b.max_conns:
                        stats['over_capacity'] += 1
                        max_over = max(max_over, b.active - b.max_conns)
                    ok, dur, out_tok = simulate_inference(rng, b, True)
                    heapq.heappush(events, (SIM_NOW + dur, seq, 'done', ('direct', b.bid)))
                    seq += 1
                    if ok:
                        b.served += 1
                        stats['direct'] += 1
                        stats['direct_served'] += 1
                        stats['succeeded'] += 1
                        stats['out_tokens'] += out_tok
                        stats['cost'] += (b.ippm * 0.7 + b.oppm * 0.3) * out_tok / 1000.0
                        success = True
                        break
                    else:
                        b.failed += 1
                        b.failures += 1
                        b.trip_cooldown()
                        stats['direct_failed'] += 1
                        stats['retries'] += 1
                        continue

            # 1. 选 client
            client = None
            backend = None
            if routing == 'cost':
                client, backend = router.pick_cheapest(model, exclude)
            elif routing == 'balanced':
                client = router.load_balance(model, exclude, LB_BALANCED_MIN_SCORE)
                if client is None:
                    backend = router.pick_direct(model, exclude)
            else:  # stability
                client = router.load_balance(model, exclude)

            if client is not None:
                exclude[client.cid] = True
                client.active += 1
                if client.active > client.max_conn:
                    stats['over_capacity'] += 1
                    max_over = max(max_over, client.active - client.max_conn)
                ok, dur, out_tok = simulate_inference(rng, client, False)
                heapq.heappush(events, (SIM_NOW + dur, seq, 'done', ('client', client.cid)))
                seq += 1
                if ok:
                    client.served += 1
                    stats['community'] += 1
                    stats['community_served'] += 1
                    stats['succeeded'] += 1
                    stats['out_tokens'] += out_tok
                    stats['cost'] += (client.ippm * 0.7 + client.oppm * 0.3) * out_tok / 1000.0
                    success = True
                    break
                else:
                    client.failed += 1
                    client.failures += 1
                    client.trip_cooldown()
                    stats['community_failed'] += 1
                    stats['retries'] += 1
                    continue
            elif backend is not None:
                exclude["direct:" + backend.bid] = True
                backend.active += 1
                if backend.active > backend.max_conns:
                    stats['over_capacity'] += 1
                    max_over = max(max_over, backend.active - backend.max_conns)
                ok, dur, out_tok = simulate_inference(rng, backend, True)
                heapq.heappush(events, (SIM_NOW + dur, seq, 'done', ('direct', backend.bid)))
                seq += 1
                if ok:
                    backend.served += 1
                    stats['direct'] += 1
                    stats['direct_served'] += 1
                    stats['succeeded'] += 1
                    stats['out_tokens'] += out_tok
                    stats['cost'] += (backend.ippm * 0.7 + backend.oppm * 0.3) * out_tok / 1000.0
                    success = True
                    break
                else:
                    backend.failed += 1
                    backend.failures += 1
                    backend.trip_cooldown()
                    stats['direct_failed'] += 1
                    stats['retries'] += 1
                    continue
            else:
                # 无候选
                break

        if not success:
            stats['unavailable'] += 1
            stats['failed'] += 1

    elapsed = time.time() - start
    return stats, total_req, elapsed, max_over


def fmt_report(routing, stats, total_req, elapsed, max_over):
    total = total_req
    fail_rate = stats['failed'] / total * 100
    over_rate = stats['over_capacity'] / total * 100
    pfx_rate = stats['prefix_hits'] / stats['prefix_total'] * 100 if stats['prefix_total'] else 0.0
    lines = []
    lines.append(f"=== routing={routing} | 总请求={total} | 模拟时长={SIM_DURATION}s | 耗时={elapsed:.1f}s ===")
    lines.append(f"  Direct 分配   : {stats['direct']:>8} ({stats['direct']/total*100:5.2f}%)")
    lines.append(f"  贡献者分配    : {stats['community']:>8} ({stats['community']/total*100:5.2f}%)")
    lines.append(f"  不可用(503)   : {stats['unavailable']:>8} ({stats['unavailable']/total*100:5.2f}%)")
    lines.append(f"  成功          : {stats['succeeded']:>8} ({stats['succeeded']/total*100:5.2f}%)")
    lines.append(f"  失败率        : {fail_rate:5.2f}%")
    lines.append(f"  超配请求      : {stats['over_capacity']:>8} ({over_rate:5.2f}%)  单节点最大超配={max_over}")
    lines.append(f"  重试次数      : {stats['retries']}")
    lines.append(f"  prefix命中率  : {pfx_rate:5.2f}%  ({stats['prefix_hits']}/{stats['prefix_total']})")
    lines.append(f"  输出tokens    : {stats['out_tokens']:,}")
    lines.append(f"  估算成本      : {stats['cost']:.2f} (元, 按单价×tokens/1000)")
    lines.append(f"  上下线        : 上线={stats['churn_up']} 下线={stats['churn_down']}")
    lines.append(f"  Direct 服务   : 成功={stats['direct_served']} 失败={stats['direct_failed']}")
    lines.append(f"  贡献者服务    : 成功={stats['community_served']} 失败={stats['community_failed']}")
    return "\n".join(lines)


def main():
    print("=" * 70)
    print("生产级负载均衡并发模拟（不启动服务，DES 离散事件）")
    print(f"消费者={N_CONSUMERS} 每消费者{REQ_PER_CONSUMER_MIN}-{REQ_PER_CONSUMER_MAX}请求 | "
          f"Direct={N_DIRECT}(MaxConns={DIRECT_MAX_CONNS}) | 贡献者={N_CONTRIBUTORS} | 时长={SIM_DURATION}s")
    print("=" * 70)
    for routing in ['stability', 'cost', 'balanced']:
        stats, total_req, elapsed, max_over = run_simulation(routing, seed=42)
        print()
        print(fmt_report(routing, stats, total_req, elapsed, max_over))
    print()
    print("=" * 70)
    print("说明：失败率含 503(不可用) + 推理失败；超配=突破节点连接上限的请求数。")
    print("=" * 70)


if __name__ == "__main__":
    main()
