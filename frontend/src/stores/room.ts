import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { BotDebugEvent, ChatMessage, Room, RoomMember, RoomReport } from '@/types'

export const useRoomStore = defineStore('room', () => {
  const room = ref<Room | null>(null)
  const self = ref<RoomMember | null>(null)
  const members = ref<RoomMember[]>([])
  const messages = ref<ChatMessage[]>([])
  const report = ref<RoomReport | null>(null)
  const wsConnected = ref(false)
  const botDebugEvents = ref<BotDebugEvent[]>([])

  function reset() {
    room.value = null
    self.value = null
    members.value = []
    messages.value = []
    report.value = null
    wsConnected.value = false
    botDebugEvents.value = []
  }

  function appendMessage(msg: ChatMessage) {
    messages.value.push(msg)
    if (messages.value.length > 300) {
      messages.value = messages.value.slice(-300)
    }
  }

  function appendBotDebugEvent(evt: BotDebugEvent) {
    botDebugEvents.value.push(evt)
    if (botDebugEvents.value.length > 200) {
      botDebugEvents.value = botDebugEvents.value.slice(-200)
    }
  }

  return {
    room,
    self,
    members,
    messages,
    report,
    wsConnected,
    botDebugEvents,
    reset,
    appendMessage,
    appendBotDebugEvent
  }
})
