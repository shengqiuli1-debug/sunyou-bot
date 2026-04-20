import { createRouter, createWebHistory } from 'vue-router'
import { useUserStore } from '@/stores/user'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'home', component: () => import('@/views/HomeView.vue') },
    { path: '/create', name: 'create', component: () => import('@/views/CreateRoomView.vue') },
    { path: '/join/:id', name: 'join', component: () => import('@/views/JoinConfirmView.vue'), props: true },
    { path: '/room/:id', name: 'room', component: () => import('@/views/RoomChatView.vue'), props: true },
    { path: '/room/:id/report', name: 'report', component: () => import('@/views/RoomReportView.vue'), props: true },
    { path: '/points', name: 'points', component: () => import('@/views/PointsView.vue') }
  ]
})

router.beforeEach(async () => {
  const user = useUserStore()
  await user.ensureGuest()
  return true
})

export default router
