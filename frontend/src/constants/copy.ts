export const COPY = {
  brand: {
    title: '损友Bot',
    slogan: '电子嫌弃机上线：专门判低质发言，负责清理群聊污染。'
  },
  roleIntro: {
    judge: '阴阳裁判：官腔判词，先定性再收刀。',
    npc: '损友 NPC：电子流氓式嫌弃，专打空心社交。',
    narrator: '冷面旁白：冷静归档社交事故，恶意最稳。'
  },
  identityIntro: {
    normal: '普通成员：正常参与聊天。',
    target: '上靶成员：Bot 更可能围绕你吐槽。',
    immune: '免疫成员：Bot 尽量不重点针对。'
  },
  targetConfirm:
    '确认后你会成为 Bot 的重点观察对象，适合熟人局。你可以随时切回 normal。',
  safeReminder:
    '边界说明：不外貌攻击、不歧视、不现实骚扰、不连续网暴。',
  homePlayGuide: [
    '1. 创建一个 5/15/30 分钟临时房间',
    '2. 分享链接给朋友进场选身份',
    '3. 实时聊天，Bot 随机插嘴吐槽',
    '4. 房间结束自动生成本局战报'
  ]
}

export const ROLE_OPTIONS = [
  { label: '阴阳裁判', value: 'judge' },
  { label: '损友 NPC', value: 'npc' },
  { label: '冷面旁白', value: 'narrator' }
]

export const FIRE_OPTIONS = [
  { label: '轻嘴', value: 'low' },
  { label: '阴阳', value: 'medium' },
  { label: '抽象/发疯', value: 'high' }
]

export const DURATION_OPTIONS = [
  { label: '5 分钟', value: 5 },
  { label: '15 分钟', value: 15 },
  { label: '30 分钟', value: 30 }
]

export const IDENTITY_OPTIONS = [
  { label: 'normal 普通成员', value: 'normal' },
  { label: 'target 上靶成员', value: 'target' },
  { label: 'immune 免疫成员', value: 'immune' }
]
