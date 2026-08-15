<script>
  import { onMount } from 'svelte'
  import { navigate } from 'svelte-routing'
  import { api } from '../lib/api.js'

  let batches = []
  let loading = true

  const typeLabels = {
    full_reduction: '满减券',
    discount: '折扣券',
    no_threshold: '无门槛券'
  }

  onMount(loadBatches)

  async function loadBatches() {
    loading = true
    try {
      batches = await api.getCouponBatches()
    } catch (e) {
      console.error('Failed to load batches:', e)
    } finally {
      loading = false
    }
  }

  function viewBatch(id) {
    navigate(`/coupons/${id}`)
  }

  function getBatchInfo(batch) {
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

  function getScopeLabel(scope) {
    switch (scope?.type) {
      case 'all': return '全场'
      case 'category': return '指定品类'
      case 'store': return '指定店铺'
      default: return '未知'
    }
  }
</script>

<div class="page-header">
  <h1>优惠券管理</h1>
</div>

<div class="card">
  <div class="card-header">
    <h2>优惠券批次列表</h2>
    <button class="btn btn-primary" on:click={() => navigate('/coupons/new')}>
      + 新建批次
    </button>
  </div>

  {#if loading}
    <p>加载中...</p>
  {:else}
    <table>
      <thead>
        <tr>
          <th>批次名称</th>
          <th>类型</th>
          <th>优惠信息</th>
          <th>适用范围</th>
          <th>有效期</th>
          <th>发放/领取/使用</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        {#each batches as batch}
          <tr>
            <td>{batch.name}</td>
            <td><span class="promo-type-badge">{typeLabels[batch.coupon_type] || batch.coupon_type}</span></td>
            <td>{getBatchInfo(batch)}</td>
            <td>{getScopeLabel(batch.scope)}</td>
            <td>
              {new Date(batch.valid_from).toLocaleDateString()}
              <br>
              ~ {new Date(batch.valid_to).toLocaleDateString()}
            </td>
            <td>
              {batch.total_quantity} / {batch.claimed_quantity} / {batch.used_quantity}
            </td>
            <td>
              <button class="btn btn-sm btn-secondary" on:click={() => viewBatch(batch.id)}>查看详情</button>
            </td>
          </tr>
        {/each}
        {#if batches.length === 0}
          <tr>
            <td colspan="7" style="text-align: center; color: #999;">暂无批次</td>
          </tr>
        {/if}
      </tbody>
    </table>
  {/if}
</div>
