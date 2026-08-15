<script>
  import { Router, Route } from 'svelte-routing'
  import Layout from './lib/Layout.svelte'
  import Dashboard from './routes/Dashboard.svelte'
  import RuleList from './routes/RuleList.svelte'
  import RuleEditor from './routes/RuleEditor.svelte'
  import RuleCalendar from './routes/RuleCalendar.svelte'
  import PriceSimulator from './routes/PriceSimulator.svelte'
  import MutexGroups from './routes/MutexGroups.svelte'
  import CouponList from './routes/CouponList.svelte'
  import CouponEditor from './routes/CouponEditor.svelte'
  import CouponDetail from './routes/CouponDetail.svelte'
  import { onMount } from 'svelte'

  let currentPath = '/'

  onMount(() => {
    currentPath = window.location.pathname
    const origPushState = history.pushState
    const origReplaceState = history.replaceState
    history.pushState = function() {
      origPushState.apply(this, arguments)
      currentPath = window.location.pathname
    }
    history.replaceState = function() {
      origReplaceState.apply(this, arguments)
      currentPath = window.location.pathname
    }
    window.addEventListener('popstate', () => {
      currentPath = window.location.pathname
    })
  })
</script>

<Router>
  <Layout {currentPath}>
    <Route path="/" component={Dashboard} />
    <Route path="/rules" component={RuleList} />
    <Route path="/rules/new"><RuleEditor id="new" /></Route>
    <Route path="/rules/:id" let:params><RuleEditor id={params.id} /></Route>
    <Route path="/calendar" component={RuleCalendar} />
    <Route path="/simulator" component={PriceSimulator} />
    <Route path="/mutex-groups" component={MutexGroups} />
    <Route path="/coupons" component={CouponList} />
    <Route path="/coupons/new"><CouponEditor id="new" /></Route>
    <Route path="/coupons/:id" let:params><CouponDetail id={params.id} /></Route>
  </Layout>
</Router>
