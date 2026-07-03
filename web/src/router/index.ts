import { createRouter, createWebHistory } from 'vue-router'
import AdminLayout from '../layouts/AdminLayout.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
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
      ],
    },
  ],
})

export default router
