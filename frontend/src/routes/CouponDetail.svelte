<script>
  import { onMount } from 'svelte'
  import { navigate } from 'svelte-routing'
  import { api } from '../lib/api.js'

  export let id

  let batch = null
  let coupons = []
  let loading = true
  let filterStatus = ''

  const statusLabels = {
    available: '未领取',
    claimed: '已领取',
    used: '已使用',
    expired: '已过期'
  }

  const typeLabels = {
    full_reduction: '满减券',
    discount: '折扣券',
    no_threshold: '无门槛券'
  }

  onMount(loadData)

  async function loadData() {
    loading = true
    try {
      batch = await api.getCouponBatch(id)
      await loadCoupons()
    } catch (e) {
      console.error('Failed to load data:', e)
    } finally {
      loading = false
    }
  }

  async function loadCoupons() {
    try {
      const data = await api.getBatchCoupons(id, filterStatus)
      coupons = data
    } catch (e) {
      console.error('Failed to load coupons:', e)
    }
  }

  function getBatchInfo() {
    if (!batch) return ''
    switch (batch.coupon_type) {
      case 'full_reduction':
        return `满${batch.threshold_amount}减${batch.discount_amount}`
      case 'discount':
        let info = `${batch.discount_rate * 10}折`
        if (batch.max_discount_amount > 0) {
          info += ` (最高减${batch.max_discount_amount})`
        }
        if (batch.threshold_amount > 0) {
          info = `满${batch.threshold_amount}可用，` + info
        }
        return info
      case 'no_threshold':
        return `无门槛减${batch.discount_amount}`
      default:
        return ''
    }
  }

  function getScopeLabel() {
    if (!batch?.scope) return '未知'
    switch (batch.scope.type) {
      case 'all': return '全场'
      case 'category': return '指定品类'
      case 'store': return '指定店铺'
      default: return '未知'
    }
  }
</script>

<div class="page-header">
  <h1>优惠券批次详情</h1>
  <button class="btn btn-secondary" on:click={() => navigate('/coupons')}>返回列表</button>
</div>

{#if loading}
  <p>加载中...</p>
{:else if batch}
  <div class="card" style="margin-bottom: 20px;">
    <div class="card-header">
      <h2>批次信息</h2>
    </div>

    <div class="info-grid">
      <div class="info-item">
        <span class="label">批次名称：</span>
        <span class="value">{batch.name}</span>
      </div>
      <div class="info-item">
        <span class="label">券类型：</span>
        <span class="value">{typeLabels[batch.coupon_type] || batch.coupon_type}</span>
      </div>
      <div class="info-item">
        <span class="label">优惠信息：</span>
        <span class="value">{getBatchInfo()}</span>
      </div>
      <div class="info-item">
        <span class="label">适用范围：</span>
        <span class="value">{getScopeLabel()}</span>
      </div>
      <div class="info-item">
        <span class="label">有效期：</span>
        <span class="value">
          {new Date(batch.valid_from).toLocaleString()}
          ~ {new Date(batch.valid_to).toLocaleString()}
        </span>
      </div>
      <div class="info-item">
        <span class="label">发放/领取/使用：</span>
        <span class="value">
          {batch.total_quantity} / {batch.claimed_quantity} / {batch.used_quantity}
        </span>
      </div>
      <div class="info-item">
        <span class="label">每人限领：</span>
        <span class="value">{batch.per_user_limit} 张</span>
      </div>
      <div class="info-item">
        <span class="label">创建人：</span>
        <span class="value">{batch.created_by || '-'}</span>
      </div>
    </div>

    {#if batch.description}
      <div style="margin-top: 15px;">
        <span class="label">描述：</span>
        <span class="value">{batch.description}</span>
      </div>
    {/if}
  </div>

  <div class="card">
    <div class="card-header">
      <h2>券码列表</h2>
      <div>
        <select class="form-control" bind:value={filterStatus} on:change={loadCoupons} style="width: auto; display: inline-block;">
          <option value="">全部状态</option>
          <option value="available">未领取</option>
          <option value="claimed">已领取</option>
          <option value="used">已使用</option>
        </select>
      </div>
    </div>

    <table>
      <thead>
        <tr>
          <th>券码</th>
          <th>状态</th>
          <th>领取用户</th>
          <th>关联订单</th>
          <th>领取时间</th>
          <th>使用时间</th>
        </tr>
      </thead>
      <tbody>
        {#each coupons as coupon}
          <tr>
            <td><code style="background: #f4f5f7; padding: 2px 6px; border-radius: 3px;">{coupon.code}</code></td>
            <td><span class="status-badge status-{coupon.status}">{statusLabels[coupon.status] || coupon.status}</span></td>
            <td>{coupon.user_id || '-'}</td>
            <td>{coupon.order_id || '-'}</td>
            <td>{coupon.claimed_at ? new Date(coupon.claimed_at).toLocaleString() : '-'}</td>
            <td>{coupon.used_at ? new Date(coupon.used_at).toLocaleString() : '-'}</td>
          </tr>
        {/each}
        {#if coupons.length === 0}
          <tr>
            <td colspan="6" style="text-align: center; color: #999;">暂无券码数据</td>
          </tr>
        {/if}
      </tbody>
    </table>
    <p style="color: #999; font-size: 12px; margin-top: 10px;">* 最多显示100条记录</p>
  </div>
{/if}

<style>
  .info-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 15px;
  }
  .info-item {
    display: flex;
    align-items: center;
  }
  .info-item .label {
    color: #6b7280;
    margin-right: 8px;
    min-width: 100px;
  }
  .info-item .value {
    font-weight: 500;
  }
  @media (max-width: 768px) {
    .info-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
