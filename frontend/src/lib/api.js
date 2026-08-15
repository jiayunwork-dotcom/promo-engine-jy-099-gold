const API_BASE = '/api'

async function request(path, options = {}) {
  const response = await fetch(`${API_BASE}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...options.headers
    },
    ...options
  })

  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`)
  }

  return response.json()
}

export const api = {
  calculatePrice: (cart) => request('/price/calculate', {
    method: 'POST',
    body: JSON.stringify(cart)
  }),

  getRules: (params = {}) => {
    const query = new URLSearchParams(params).toString()
    return request(`/rules${query ? '?' + query : ''}`)
  },

  getRule: (id) => request(`/rules/${id}`),

  createRule: (rule) => request('/rules', {
    method: 'POST',
    body: JSON.stringify(rule)
  }),

  updateRule: (id, rule) => request(`/rules/${id}`, {
    method: 'PUT',
    body: JSON.stringify(rule)
  }),

  deleteRule: (id) => request(`/rules/${id}`, {
    method: 'DELETE'
  }),

  updateRuleStatus: (id, status) => request(`/rules/${id}/status`, {
    method: 'PATCH',
    body: JSON.stringify({ status })
  }),

  getRuleVersions: (id) => request(`/rules/${id}/versions`),

  rollbackVersion: (id, version) => request(`/rules/${id}/versions/${version}/rollback`, {
    method: 'POST'
  }),

  detectConflicts: (rule) => request('/rules/detect-conflicts', {
    method: 'POST',
    body: JSON.stringify(rule)
  }),

  estimateEffect: (rule) => request('/rules/estimate-effect', {
    method: 'POST',
    body: JSON.stringify(rule)
  }),

  getMutexGroups: () => request('/mutex-groups'),

  getMutexGroup: (id) => request(`/mutex-groups/${id}`),

  createMutexGroup: (group) => request('/mutex-groups', {
    method: 'POST',
    body: JSON.stringify(group)
  }),

  updateMutexGroup: (id, group) => request(`/mutex-groups/${id}`, {
    method: 'PUT',
    body: JSON.stringify(group)
  }),

  deleteMutexGroup: (id) => request(`/mutex-groups/${id}`, {
    method: 'DELETE'
  }),

  addRuleToMutexGroup: (groupId, ruleId) => request(`/mutex-groups/${groupId}/rules/${ruleId}`, {
    method: 'POST'
  }),

  removeRuleFromMutexGroup: (groupId, ruleId) => request(`/mutex-groups/${groupId}/rules/${ruleId}`, {
    method: 'DELETE'
  }),

  getCouponBatches: () => request('/coupon-batches'),

  getCouponBatch: (id) => request(`/coupon-batches/${id}`),

  createCouponBatch: (batch) => request('/coupon-batches', {
    method: 'POST',
    body: JSON.stringify(batch)
  }),

  getBatchCoupons: (id, status = '') => {
    const query = status ? `?status=${status}` : ''
    return request(`/coupon-batches/${id}/coupons${query}`)
  },

  claimCoupon: (data) => request('/coupons/claim', {
    method: 'POST',
    body: JSON.stringify(data)
  }),

  getUserCoupons: (userId, status = '') => {
    const query = status ? `?status=${status}` : ''
    return request(`/coupons/user/${userId}${query}`)
  },

  useCoupon: (data) => request('/coupons/use', {
    method: 'POST',
    body: JSON.stringify(data)
  }),

  returnCoupon: (data) => request('/coupons/return', {
    method: 'POST',
    body: JSON.stringify(data)
  })
}
