import type { BotRole } from '@/types'

export interface BotRoleMeta {
  role: BotRole
  name: string
  bio: string
  identity: string
  duty: string
  avatarText: string
  styleDesc: string
}

export const BOT_ROLE_META: Record<BotRole, BotRoleMeta> = {
  judge: {
    role: 'judge',
    name: '阴阳裁判',
    bio: '房间秩序鉴定员',
    identity: '房间角色 / 特殊成员',
    duty: '负责定性低质发言、收尾和节奏压舱。',
    avatarText: '判',
    styleDesc: '高位冷嘲，克制压人。'
  },
  npc: {
    role: 'npc',
    name: '损友 NPC',
    bio: '低质发言巡查员',
    identity: '房间角色 / 特殊成员',
    duty: '负责高频接话、抓破绽、现场补刀。',
    avatarText: '损',
    styleDesc: '口语快刀，像群里真人嘴碎损友。'
  },
  narrator: {
    role: 'narrator',
    name: '冷面旁白',
    bio: '现场事故记录员',
    identity: '房间角色 / 特殊成员',
    duty: '负责冲突后的冷观察补刀和局势总结。',
    avatarText: '述',
    styleDesc: '冷静观察，稳态轻蔑。'
  }
}

export function normalizeBotRole(v?: string): BotRole {
  if (v === 'judge' || v === 'npc' || v === 'narrator') return v
  return 'npc'
}

export function inferBotRoleFromNickname(name?: string): BotRole {
  const raw = String(name || '').toLowerCase()
  if (raw.includes('阴阳裁判') || raw.includes('judge')) return 'judge'
  if (raw.includes('冷面旁白') || raw.includes('narrator')) return 'narrator'
  return 'npc'
}

export function resolveBotRole(role?: string, nickname?: string): BotRole {
  if (role === 'judge' || role === 'npc' || role === 'narrator') return role
  return inferBotRoleFromNickname(nickname)
}
