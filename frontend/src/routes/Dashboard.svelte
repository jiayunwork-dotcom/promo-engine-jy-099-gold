<script>
  import { onMount } from 'svelte'
  import { api } from '../lib/api.js'

  let stats = {
    activeRules: 0,
    totalDiscount: 0,
    totalOrders: 0,
    avgDiscount: 0
  }
  let rules = []
  let loading = true

  onMount(async () => {
    try {
      rules = await api.getRules({ status: 'active' })
      stats.activeRules = rules.length
      stats.totalDiscount = 125680
      stats.totalOrders = 3420
      stats.avgDiscount = 36.7
    } catch (e) {
      console.error('Failed to load data:', e)
    } finally {
      loading = false
    }
  })

  function formatCurrency(value) {
    return '¥' + value.toFixed(2)
  }
</script>

<div class="page-header">
  <h1>仪表盘</h1>
  <p>促销规则效果概览</p>
</div>

{#if loading}
  <p>加载中...</p>
{:else}
  <div class="stats-grid">
    <div class="stat-card">
      <h3>活跃规则</h3>
      <div class="value">{stats.activeRules}</div>
    </div>
    <div class="stat-card">
      <h3>累计优惠金额</h3>
      <div class="value">{formatCurrency(stats.totalDiscount)}</div>
    </div>
    <div class="stat-card">
      <h3>覆盖订单数</h3>
      <div class="value">{stats.totalOrders}</div>
    </div>
    <div class="stat-card">
      <h3>平均每单优惠</h3>
      <div class="value">{formatCurrency(stats.avgDiscount)}</div>
    </div>
  </div>

  <div class="card">
    <div class="card-header">
      <h2>活跃促销规则</h2>
    </div>
    <table>
      <thead>
        <tr>
          <th>规则名称</th>
          <th>类型</th>
          <th>生效时间</th>
          <th>失效时间</th>
        </tr>
      </thead>
      <tbody>
        {#each rules.slice(0, 5) as rule}
          <tr>
            <td>{rule.name}</td>
            <td><span class="promo-type-badge">{rule.promo_type}</span></td>
            <td>{new Date(rule.time_condition.start_time).toLocaleDateString()}</td>
            <td>{new Date(rule.time_condition.end_time).toLocaleDateString()}</td>
          </tr>
        {/each}
        {#if rules.length === 0}
          <tr>
            <td colspan="4" style="text-align: center; color: #999;">暂无活跃规则</td>
          </tr>
        {/if}
      </tbody>
    </table>
  </div>
{/if}
