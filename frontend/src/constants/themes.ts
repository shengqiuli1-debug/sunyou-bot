export type ThemeId =
  | 'night-blue'
  | 'warm-glow'
  | 'sunny-pop'
  | 'mint-breeze'
  | 'sea-salt'
  | 'coral-party'
  | 'mist-violet'

export interface ThemeOption {
  id: ThemeId
  name: string
  tone: string
  description: string
  mode: 'light' | 'dark'
  swatches: [string, string, string, string]
}

export const DEFAULT_THEME_ID: ThemeId = 'night-blue'
export const THEME_STORAGE_KEY = 'sunyou_theme'

export const THEME_OPTIONS: ThemeOption[] = [
  {
    id: 'night-blue',
    name: '夜幕蓝',
    tone: '夜间 / 科技 / 柔和',
    description: '深蓝灰底色，聊天集中不刺眼。',
    mode: 'dark',
    swatches: ['#141922', '#1C2430', '#7C9CFF', '#8EE0C2']
  },
  {
    id: 'warm-glow',
    name: '暖光温馨',
    tone: '温暖 / 放松 / 熟人局',
    description: '像暖灯下的夜聊，轻松不压抑。',
    mode: 'light',
    swatches: ['#FFF6EE', '#FFE7D3', '#E78A4E', '#C96B5C']
  },
  {
    id: 'sunny-pop',
    name: '阳光活力黄',
    tone: '明亮 / 热闹 / 聚会',
    description: '高活力但保留可读性，适合整活。',
    mode: 'light',
    swatches: ['#FFF9DC', '#FFE98A', '#E0A100', '#FFB703']
  },
  {
    id: 'mint-breeze',
    name: '薄荷清新绿',
    tone: '清爽 / 舒缓 / 白天',
    description: '空气感更强，长时间阅读更轻松。',
    mode: 'light',
    swatches: ['#EEF9F1', '#D3ECD9', '#49A86E', '#73C08E']
  },
  {
    id: 'sea-salt',
    name: '海盐清澈蓝',
    tone: '冷静 / 干净 / 理性',
    description: '整体克制清透，信息层级更清晰。',
    mode: 'light',
    swatches: ['#EEF6FF', '#D2E7FF', '#3D7EDB', '#67B4FF']
  },
  {
    id: 'coral-party',
    name: '珊瑚活力橙红',
    tone: '年轻 / 派对 / 冲击',
    description: '更有现场感，适合热闹房间。',
    mode: 'light',
    swatches: ['#FFF2EC', '#FFD4C4', '#E56B4E', '#F28C64']
  },
  {
    id: 'mist-violet',
    name: '雾紫安静',
    tone: '安静 / 柔和 / 夜聊',
    description: '带一点神秘感，适合慢节奏聊天。',
    mode: 'light',
    swatches: ['#F6F1FF', '#E2D7FA', '#8A6FD1', '#B39DFF']
  }
]

const themeMap = new Map(THEME_OPTIONS.map((t) => [t.id, t]))

export function resolveThemeId(id?: string | null): ThemeId {
  if (id && themeMap.has(id as ThemeId)) {
    return id as ThemeId
  }
  return DEFAULT_THEME_ID
}

export function getThemeOption(id?: string | null): ThemeOption {
  return themeMap.get(resolveThemeId(id)) || THEME_OPTIONS[0]
}
