"""
智能负载均衡算法模拟测试
生成几千组随机 client 数据，验证多维度加权评分算法的正确性、区分度、稳定性。
"""
import random
import statistics
import time

# ============ 算法实现（与 Go 方案一致） ============

# 权重配置
# 会员等级权重提高：不同贡献者会员（normal/vip/svip）在相同模型时被负载的程度和获益应不同。
# VIP/SVIP 贡献者应被优先选中，获得更多流量和收益。
WEIGHTS = {
    'capacity': 0.20,
    'membership': 0.25,  # 提高：会员等级是核心差异化维度
    'latency': 0.20,
    'failure': 0.15,
    'stability': 0.10,
    'service': 0.10,
}

MAXLATENCE = 30000  # 与 public.MAXLATENCE 一致
TARGET_ONLINE_SEC = 3600  # 目标在线时长（1小时）
TARGET_TOKENS_PER_HOUR = 100000  # 目标产能（token/小时）

# 会员等级分：拉大差距，让不同等级获益明显不同
# normal=0.2, vip=0.6, svip=1.0
MEMBERSHIP_SCORES = {'normal': 0.2, 'vip': 0.6, 'svip': 1.0}


def membership_score(membership):
    """会员等级分：normal=0.2, vip=0.6, svip=1.0（拉大差距）"""
    return MEMBERSHIP_SCORES.get(membership, 0.2)


def capacity_score(max_conn, active_conn):
    """可用连接数分：SVIP(-1)无限=1.0，否则 可用/上限"""
    if max_conn < 0:
        return 1.0
    if max_conn <= 0:
        return 0.0
    return max(0.0, min(1.0, (max_conn - active_conn) / max_conn))


# 延迟平滑参数（EMA 指数移动平均）
EMA_ALPHA = 0.3  # 平滑系数：越大越跟随瞬时值，越小越平滑


def ema_update(prev_ema, new_sample, alpha=EMA_ALPHA):
    """指数移动平均更新：ema = alpha*new + (1-alpha)*prev"""
    if prev_ema is None:
        return new_sample
    return alpha * new_sample + (1 - alpha) * prev_ema


def latency_score(latency, min_latency, max_latency):
    """延迟分：基于 MAXLATENCE 的绝对映射。
    使用绝对延迟而非纯相对归一化，避免中等延迟（如300ms）因相对比较得0分，
    从而让会员等级优势能正常体现。延迟越低分越高。
    传入的 latency 应为 EMA 平滑后的值。"""
    return max(0.0, min(1.0, 1 - latency / MAXLATENCE))


def failure_score(recent_failures):
    """失败率分：1/(1+failures)，失败越多越低"""
    return 1.0 / (1.0 + recent_failures)


def stability_score(total_online_sec, disconnect_count):
    """在线稳定性分：平均在线时长 / 目标时长"""
    avg_online = total_online_sec / (disconnect_count + 1)
    return max(0.0, min(1.0, avg_online / TARGET_ONLINE_SEC))


def service_score(total_tokens, total_online_sec):
    """服务等级分：贡献token/小时 / 目标产能"""
    if total_online_sec <= 0:
        return 0.0
    tokens_per_hour = total_tokens / (total_online_sec / 3600.0)
    return max(0.0, min(1.0, tokens_per_hour / TARGET_TOKENS_PER_HOUR))


def compute_score(client, min_latency, max_latency):
    """计算单个 client 的综合评分"""
    s = (
        WEIGHTS['capacity'] * capacity_score(client['max_conn'], client['active_conn'])
        + WEIGHTS['membership'] * membership_score(client['membership'])
        + WEIGHTS['latency'] * latency_score(client['latency'], min_latency, max_latency)
        + WEIGHTS['failure'] * failure_score(client['recent_failures'])
        + WEIGHTS['stability'] * stability_score(client['total_online_sec'], client['disconnect_count'])
        + WEIGHTS['service'] * service_score(client['total_tokens'], client['total_online_sec'])
    )
    return s


def pick_smart(clients, jitter=True):
    """选择综合评分最高的 client，可选加入随机扰动防抖"""
    if not clients:
        return None
    min_latency = min(c['latency'] for c in clients)
    max_latency = max(c['latency'] for c in clients)
    best = None
    best_score = -1
    for c in clients:
        score = compute_score(c, min_latency, max_latency)
        if jitter:
            score *= random.uniform(0.95, 1.05)  # ±5% 扰动
        if score > best_score:
            best_score = score
            best = c
    return best


# ============ 测试数据生成 ============

def gen_client(quality='random'):
    """生成一个随机 client。quality: random/good/bad"""
    if quality == 'good':
        membership = random.choice(['vip', 'svip'])
        max_conn = 10 if membership == 'vip' else -1
        active_conn = random.randint(0, max(0, max_conn // 3))
        latency = random.randint(20, 200)
        failures = random.randint(0, 2)
        online_sec = random.randint(3600 * 24, 3600 * 24 * 30)
        disconnect = random.randint(0, 3)
        tokens = random.randint(100000, 5000000)
    elif quality == 'bad':
        membership = 'normal'
        max_conn = 3
        active_conn = random.randint(2, 3)
        latency = random.randint(5000, 25000)
        failures = random.randint(5, 30)
        online_sec = random.randint(60, 3600)
        disconnect = random.randint(10, 50)
        tokens = random.randint(0, 10000)
    else:  # random
        membership = random.choice(['normal', 'vip', 'svip'])
        max_conn = {'normal': 3, 'vip': 10, 'svip': -1}[membership]
        active_conn = random.randint(0, max(0, max_conn))
        latency = random.randint(20, 25000)
        failures = random.randint(0, 20)
        online_sec = random.randint(60, 3600 * 24 * 30)
        disconnect = random.randint(0, 30)
        tokens = random.randint(0, 3000000)

    return {
        'id': f'client_{random.randint(1000, 9999)}',
        'membership': membership,
        'max_conn': max_conn,
        'active_conn': active_conn,
        'latency': latency,
        'recent_failures': failures,
        'total_online_sec': online_sec,
        'disconnect_count': disconnect,
        'total_tokens': tokens,
    }


# ============ 测试 ============

def test_correctness():
    """正确性：分数在 [0,1] 范围，无异常"""
    print("=== 测试1: 正确性（分数范围） ===")
    ok = True
    for _ in range(5000):
        clients = [gen_client() for _ in range(random.randint(3, 20))]
        min_lat = min(c['latency'] for c in clients)
        max_lat = max(c['latency'] for c in clients)
        for c in clients:
            s = compute_score(c, min_lat, max_lat)
            if not (0 <= s <= 1.0001):
                ok = False
                print(f"  越界: {c['id']} score={s}")
    print(f"  结果: {'PASS' if ok else 'FAIL'}")


def test_discrimination():
    """区分度：高质量 client 应稳定胜出"""
    print("=== 测试2: 区分度（好 vs 坏） ===")
    good_wins = 0
    total = 3000
    for _ in range(total):
        clients = [gen_client('good') for _ in range(3)] + [gen_client('bad') for _ in range(3)]
        random.shuffle(clients)
        winner = pick_smart(clients, jitter=False)
        # 判断 winner 是否为 good（通过特征判断）
        is_good = winner['latency'] < 1000 and winner['recent_failures'] <= 2 and winner['disconnect_count'] <= 3
        if is_good:
            good_wins += 1
    rate = good_wins / total
    print(f"  高质量 client 胜出率: {rate:.2%}")
    print(f"  结果: {'PASS' if rate > 0.95 else 'FAIL'}")


def test_stability():
    """稳定性：相同输入（无扰动）得到相同结果"""
    print("=== 测试3: 稳定性（无扰动确定性） ===")
    ok = True
    for _ in range(2000):
        clients = [gen_client() for _ in range(random.randint(3, 20))]
        w1 = pick_smart(clients, jitter=False)
        w2 = pick_smart(clients, jitter=False)
        if w1['id'] != w2['id']:
            ok = False
            print(f"  不稳定: {w1['id']} vs {w2['id']}")
            break
    print(f"  结果: {'PASS' if ok else 'FAIL'}")


def test_bad_never_wins():
    """极端场景：全坏 client 中，应选出综合评分最高的；好+坏混合，坏不应胜出"""
    print("=== 测试4: 极端场景 ===")
    # 场景A：全坏，应选出综合评分最高的
    correct = 0
    for _ in range(2000):
        clients = [gen_client('bad') for _ in range(5)]
        winner = pick_smart(clients, jitter=False)
        min_lat = min(c['latency'] for c in clients)
        max_lat = max(c['latency'] for c in clients)
        best_score = max(compute_score(c, min_lat, max_lat) for c in clients)
        if abs(compute_score(winner, min_lat, max_lat) - best_score) < 1e-9:
            correct += 1
    print(f"  全坏场景选出综合分最高: {correct/2000:.2%}")
    print(f"  结果: {'PASS' if correct/2000 > 0.99 else 'FAIL'}")

    # 场景B：好+坏混合，坏 client 不应胜出
    bad_wins = 0
    for _ in range(2000):
        clients = [gen_client('good') for _ in range(3)] + [gen_client('bad') for _ in range(3)]
        random.shuffle(clients)
        winner = pick_smart(clients, jitter=False)
        is_bad = winner['latency'] >= 5000 or winner['recent_failures'] > 5 or winner['disconnect_count'] > 10
        if is_bad:
            bad_wins += 1
    print(f"  好+坏混合中坏 client 胜出率: {bad_wins/2000:.2%}")
    print(f"  结果: {'PASS' if bad_wins/2000 < 0.01 else 'FAIL'}")


def test_weight_sensitivity():
    """权重敏感性：调高延迟权重后，低延迟 client 更易胜出"""
    print("=== 测试5: 权重敏感性 ===")
    global WEIGHTS
    orig = dict(WEIGHTS)
    # 调高延迟权重
    WEIGHTS = {'capacity': 0.1, 'membership': 0.05, 'latency': 0.6,
               'failure': 0.1, 'stability': 0.1, 'service': 0.05}
    low_latency_wins = 0
    total = 2000
    for _ in range(total):
        clients = [gen_client() for _ in range(5)]
        # 强制一个低延迟 client
        clients[0]['latency'] = 30
        clients[0]['recent_failures'] = 0
        winner = pick_smart(clients, jitter=False)
        if winner['id'] == clients[0]['id']:
            low_latency_wins += 1
    WEIGHTS = orig
    print(f"  高延迟权重下低延迟胜出率: {low_latency_wins/total:.2%}")
    print(f"  结果: {'PASS' if low_latency_wins/total > 0.8 else 'FAIL'}")


def test_performance():
    """性能：几千组数据的总耗时"""
    print("=== 测试6: 性能 ===")
    start = time.time()
    total_clients = 0
    for _ in range(5000):
        clients = [gen_client() for _ in range(random.randint(3, 20))]
        total_clients += len(clients)
        pick_smart(clients, jitter=True)
    elapsed = time.time() - start
    print(f"  处理 {total_clients} 个 client，耗时 {elapsed:.3f}s")
    print(f"  平均每个 client: {elapsed/total_clients*1e6:.2f}µs")
    print(f"  结果: {'PASS' if elapsed < 5 else 'FAIL'}")


def test_membership_priority():
    """会员等级专项：相同模型下，VIP/SVIP 贡献者应被优先选中，获益不同"""
    print("=== 测试7: 会员等级优先级 ===")

    # 场景A：其他维度完全相同，仅会员等级不同 → 高等级应稳定胜出
    print("  [A] 其他维度相同，仅会员等级不同")
    wins = {'normal': 0, 'vip': 0, 'svip': 0}
    total = 3000
    for _ in range(total):
        base = gen_client()
        clients = []
        for m in ['normal', 'vip', 'svip']:
            c = dict(base)
            c['membership'] = m
            c['max_conn'] = {'normal': 3, 'vip': 10, 'svip': -1}[m]
            c['active_conn'] = 0  # 全部空闲，公平比较
            c['latency'] = 100  # 相同延迟
            c['recent_failures'] = 0
            c['disconnect_count'] = 0
            c['total_online_sec'] = 3600 * 24
            c['total_tokens'] = 500000
            clients.append(c)
        winner = pick_smart(clients, jitter=False)
        wins[winner['membership']] += 1
    print(f"   胜出分布: normal={wins['normal']/total:.1%}, "
          f"vip={wins['vip']/total:.1%}, svip={wins['svip']/total:.1%}")
    print(f"   结果: {'PASS' if wins['svip']/total > 0.99 else 'FAIL'}")

    # 场景B：SVIP 延迟略高，仍应胜出（会员等级权重 > 延迟权重）
    print("  [B] SVIP 延迟略高(300ms) vs VIP 低延迟(50ms)")
    svip_wins = 0
    total = 3000
    for _ in range(total):
        clients = [
            {'id': 'vip_c', 'membership': 'vip', 'max_conn': 10, 'active_conn': 0,
             'latency': 50, 'recent_failures': 0, 'total_online_sec': 3600 * 24,
             'disconnect_count': 0, 'total_tokens': 500000},
            {'id': 'svip_c', 'membership': 'svip', 'max_conn': -1, 'active_conn': 0,
             'latency': 300, 'recent_failures': 0, 'total_online_sec': 3600 * 24,
             'disconnect_count': 0, 'total_tokens': 500000},
        ]
        winner = pick_smart(clients, jitter=False)
        if winner['id'] == 'svip_c':
            svip_wins += 1
    print(f"   SVIP 胜出率: {svip_wins/total:.2%}")
    print(f"   结果: {'PASS' if svip_wins/total > 0.9 else 'FAIL'}")

    # 场景C：VIP 与 normal 竞争，VIP 应稳定胜出
    print("  [C] VIP vs normal（其他相同）")
    vip_wins = 0
    total = 3000
    for _ in range(total):
        clients = [
            {'id': 'normal_c', 'membership': 'normal', 'max_conn': 3, 'active_conn': 0,
             'latency': 100, 'recent_failures': 0, 'total_online_sec': 3600 * 24,
             'disconnect_count': 0, 'total_tokens': 500000},
            {'id': 'vip_c', 'membership': 'vip', 'max_conn': 10, 'active_conn': 0,
             'latency': 100, 'recent_failures': 0, 'total_online_sec': 3600 * 24,
             'disconnect_count': 0, 'total_tokens': 500000},
        ]
        winner = pick_smart(clients, jitter=False)
        if winner['id'] == 'vip_c':
            vip_wins += 1
    print(f"   VIP 胜出率: {vip_wins/total:.2%}")
    print(f"   结果: {'PASS' if vip_wins/total > 0.99 else 'FAIL'}")

    # 场景D：流量分配比例（获益差异）——统计各等级被选中的占比
    print("  [D] 混合场景流量分配（获益差异）")
    counts = {'normal': 0, 'vip': 0, 'svip': 0}
    total = 5000
    for _ in range(total):
        clients = [gen_client() for _ in range(6)]
        winner = pick_smart(clients, jitter=True)
        counts[winner['membership']] += 1
    print(f"   流量分配: normal={counts['normal']/total:.1%}, "
          f"vip={counts['vip']/total:.1%}, svip={counts['svip']/total:.1%}")
    # 期望：svip 占比 > vip > normal（会员等级权重生效）
    ok = counts['svip'] >= counts['vip'] >= counts['normal']
    print(f"   结果: {'PASS' if ok else 'FAIL'}")


def test_latency_smoothing():
    """延迟平滑专项：EMA 应抑制瞬时抖动，且能跟踪趋势"""
    print("=== 测试8: 延迟平滑（EMA） ===")

    # 场景A：抖动抑制——一个稳定低延迟 client 不应因单次抖动而失分
    print("  [A] 抖动抑制：稳定低延迟 vs 偶发抖动")
    # 稳定 client：始终 100ms
    stable = {'id': 'stable', 'membership': 'vip', 'max_conn': 10, 'active_conn': 0,
              'recent_failures': 0, 'total_online_sec': 3600 * 24,
              'disconnect_count': 0, 'total_tokens': 500000}
    # 抖动 client：多数 100ms，但偶发 20000ms 尖峰
    jittery = {'id': 'jittery', 'membership': 'vip', 'max_conn': 10, 'active_conn': 0,
               'recent_failures': 0, 'total_online_sec': 3600 * 24,
               'disconnect_count': 0, 'total_tokens': 500000}

    # 模拟 100 次心跳，抖动 client 每 10 次出现一次 20000ms 尖峰
    stable_ema = None
    jittery_ema = None
    stable_ema_series = []
    jittery_ema_series = []
    for i in range(100):
        stable_sample = 100
        jittery_sample = 20000 if i % 10 == 0 else 100
        stable_ema = ema_update(stable_ema, stable_sample)
        jittery_ema = ema_update(jittery_ema, jittery_sample)
        stable_ema_series.append(stable_ema)
        jittery_ema_series.append(jittery_ema)

    # 计算原始瞬时值的波动 vs EMA 值的波动
    jittery_raw = [20000 if i % 10 == 0 else 100 for i in range(100)]
    raw_std = statistics.pstdev(jittery_raw)
    ema_std = statistics.pstdev(jittery_ema_series)
    print(f"   抖动client原始瞬时延迟标准差: {raw_std:.1f}ms")
    print(f"   抖动client EMA平滑后标准差: {ema_std:.1f}ms")
    print(f"   平滑后最终EMA延迟: {jittery_ema:.1f}ms")
    print(f"   结果: {'PASS' if ema_std < raw_std * 0.5 else 'FAIL'}")

    # 场景B：趋势跟踪——延迟持续升高时 EMA 应跟随
    print("  [B] 趋势跟踪：延迟从 100ms 逐步升到 5000ms")
    ema = None
    for i in range(50):
        sample = 100 + (i / 49) * 4900  # 100 -> 5000
        ema = ema_update(ema, sample)
    print(f"   最终EMA延迟: {ema:.1f}ms（真实值5000ms）")
    # EMA 应接近真实值（跟随趋势）
    ok = ema > 4000
    print(f"   结果: {'PASS' if ok else 'FAIL'}")

    # 场景C：平滑后评分稳定性——抖动 client 的评分不应剧烈波动
    print("  [C] 平滑后评分稳定性")
    scores = []
    for ema_val in jittery_ema_series:
        jittery['latency'] = ema_val
        scores.append(compute_score(jittery, 0, MAXLATENCE))
    score_std = statistics.pstdev(scores)
    print(f"   抖动client平滑后评分标准差: {score_std:.4f}")
    print(f"   结果: {'PASS' if score_std < 0.02 else 'FAIL'}")


if __name__ == '__main__':
    random.seed(42)
    test_correctness()
    test_discrimination()
    test_stability()
    test_bad_never_wins()
    test_weight_sensitivity()
    test_performance()
    test_membership_priority()
    test_latency_smoothing()
    print("\n全部测试完成")
