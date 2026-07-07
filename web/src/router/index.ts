import { createRouter, createWebHistory } from 'vue-router'
import AdminLayout from '../layouts/AdminLayout.vue'
import { getToken, redirectToPortal, ensureSession, clearToken } from '../utils/auth'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/auth/callback',
      name: 'AuthCallback',
      component: () => import('../views/AuthCallback.vue'),
      meta: { public: true },
    },
    {
      path: '/auth/logout',
      name: 'AuthLogout',
      component: () => import('../views/AuthLogout.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: AdminLayout,
      redirect: '/dashboard',
      children: [
        {
          path: 'dashboard',
          name: 'Dashboard',
          component: () => import('../views/Dashboard.vue'),
          meta: { title: '工作台' },
        },
        {
          path: 'kdzs-accounts',
          name: 'KdzsAccountList',
          component: () => import('../views/kdzs/KdzsAccountList.vue'),
          meta: { title: '账号管理' },
        },
        {
          path: 'shops',
          name: 'ShopList',
          component: () => import('../views/shop/ShopList.vue'),
          meta: { title: '店铺管理' },
        },
        {
          path: 'factories',
          name: 'FactoryList',
          component: () => import('../views/factory/FactoryList.vue'),
          meta: { title: '厂家管理' },
        },
        {
          path: 'orders',
          name: 'OrderList',
          component: () => import('../views/order/OrderList.vue'),
          meta: { title: '订单列表' },
        },
        {
          path: 'refunds',
          name: 'RefundList',
          component: () => import('../views/refund/RefundList.vue'),
          meta: { title: '售后列表' },
        },
        {
          path: 'return-exchanges',
          name: 'ReturnExchangeList',
          component: () => import('../views/return-exchange/ReturnExchangeList.vue'),
          meta: { title: '退换货管理' },
        },
        {
          path: 'notifications',
          name: 'NotificationSettings',
          component: () => import('../views/notification/NotificationSettings.vue'),
          meta: { title: '通知管理' },
        },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  if (to.meta.public) return true
  if (!getToken()) {
    redirectToPortal()
    return false
  }
  const ok = await ensureSession()
  if (!ok) {
    clearToken()
    redirectToPortal()
    return false
  }
  return true
})

export default router
