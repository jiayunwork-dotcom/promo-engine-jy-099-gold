<script>
  import { onMount } from 'svelte'
  import { navigate } from 'svelte-routing'
  import { api } from '../lib/api.js'

  export let id = 'new'

  let batch = {
    name: '',
    description: '',
    coupon_type: 'full_reduction',
    discount_amount: 0,
    discount_rate: 0.9,
    max_discount_amount: 0,
    threshold_amount: 0,
    scope: { type: 'all' },
    valid_from: '',
    valid_to: '',
    total_quantity: 100,
    per_user_limit: 1,
    created_by: 'admin'
  }

  let loading = false

  onMount(() => {
    const now = new Date()
    const nextMonth = new Date(now.getTime() + 30 * 24 * 60 * 60 * 1000)
    batch.valid_from = now.toISOString().slice(0, 16)
    batch.valid_to = nextMonth.toISOString().slice(0, 16)
  })

  async function save() {
    if (!batch.name) {
      alert('请输入批次名称')
      return
    }
    if (batch.total_quantity <= 0) {
      alert('发放总量必须大于0')
      return
    }

    loading = true
    try {
      await api.createCouponBatch(batch)
      alert('创建成功')
      navigate('/coupons')
    } catch (e) {
      alert('创建失败')
    } finally {
      loading = false
    }
  }
</script>

<div class="page-header">
  <h1>新建优惠券批次</h1>
  <button class="btn btn-secondary" on:click={() => navigate('/coupons')}>返回列表</button>
</div>

<div class="card">
  <div class="card-header">
    <h2>基本信息</h2>
  </div>

  <div class="form-row">
    <div class="form-group" style="flex: 2;">
      <label>批次名称 *</label>
      <input type="text" class="form-control" bind:value={batch.name} placeholder="请输入批次名称">
    </div>
    <div class="form-group">
      <label>创建人</label>
      <input type="text" class="form-control" bind:value={batch.created_by}>
    </div>
  </div>

  <div class="form-group">
    <label>描述</label>
    <textarea class="form-control" bind:value={batch.description} rows="2"></textarea>
  </div>

  <div class="form-row">
    <div class="form-group">
      <label>券类型 *</label>
      <select class="form-control" bind:value={batch.coupon_type}>
        <option value="full_reduction">满减券</option>
        <option value="discount">折扣券</option>
        <option value="no_threshold">无门槛券</option>
      </select>
    </div>
    <div class="form-group">
      <label>发放总量 *</label>
      <input type="number" class="form-control" bind:value={batch.total_quantity} min="1">
    </div>
    <div class="form-group">
      <label>每人限领</label>
      <input type="number" class="form-control" bind:value={batch.per_user_limit} min="1">
    </div>
  </div>

  {#if batch.coupon_type === 'full_reduction'}
    <div class="form-row">
      <div class="form-group">
        <label>使用门槛（元）</label>
        <input type="number" class="form-control" bind:value={batch.threshold_amount} min="0" step="0.01">
      </div>
      <div class="form-group">
        <label>抵扣金额（元）*</label>
        <input type="number" class="form-control" bind:value={batch.discount_amount} min="0" step="0.01">
      </div>
    </div>
  {/if}

  {#if batch.coupon_type === 'discount'}
    <div class="form-row">
      <div class="form-group">
        <label>使用门槛（元）</label>
        <input type="number" class="form-control" bind:value={batch.threshold_amount} min="0" step="0.01">
      </div>
      <div class="form-group">
        <label>折扣率（如0.9表示9折）*</label>
        <input type="number" class="form-control" bind:value={batch.discount_rate} min="0" max="1" step="0.01">
      </div>
      <div class="form-group">
        <label>最高优惠金额（元，0表示不限制）</label>
        <input type="number" class="form-control" bind:value={batch.max_discount_amount} min="0" step="0.01">
      </div>
    </div>
  {/if}

  {#if batch.coupon_type === 'no_threshold'}
    <div class="form-row">
      <div class="form-group">
        <label>抵扣金额（元）*</label>
        <input type="number" class="form-control" bind:value={batch.discount_amount} min="0" step="0.01">
      </div>
    </div>
  {/if}

  <div class="form-row">
    <div class="form-group">
      <label>生效时间 *</label>
      <input type="datetime-local" class="form-control" bind:value={batch.valid_from}>
    </div>
    <div class="form-group">
      <label>失效时间 *</label>
      <input type="datetime-local" class="form-control" bind:value={batch.valid_to}>
    </div>
  </div>

  <div class="form-row">
    <div class="form-group">
      <label>适用范围</label>
      <select class="form-control" bind:value={batch.scope.type}>
        <option value="all">全场</option>
        <option value="category">指定品类</option>
        <option value="store">指定店铺</option>
      </select>
    </div>
    {#if batch.scope.type === 'category'}
      <div class="form-group" style="flex: 2;">
        <label>品类ID（多个用逗号分隔）</label>
        <input type="text" class="form-control" on:change={(e) => batch.scope.category_ids = e.target.value.split(',').map(Number).filter(n => n)}>
      </div>
    {/if}
    {#if batch.scope.type === 'store'}
      <div class="form-group" style="flex: 2;">
        <label>店铺ID（多个用逗号分隔）</label>
        <input type="text" class="form-control" on:change={(e) => batch.scope.store_ids = e.target.value.split(',').map(Number).filter(n => n)}>
      </div>
    {/if}
  </div>

  <div style="margin-top: 20px;">
    <button class="btn btn-primary" on:click={save} disabled={loading}>
      {loading ? '创建中...' : '创建批次'}
    </button>
  </div>
</div>
