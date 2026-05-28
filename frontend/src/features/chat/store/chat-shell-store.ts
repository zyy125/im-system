import { create } from 'zustand'

type PrimaryView = 'messages' | 'contacts'
type ContactsView = 'friends' | 'groups'

interface ChatShellState {
  primaryView: PrimaryView
  contactsView: ContactsView
  activeConversationId: number | null
  setPrimaryView: (view: PrimaryView) => void
  setContactsView: (view: ContactsView) => void
  setActiveConversationId: (conversationId: number | null) => void
}

export const useChatShellStore = create<ChatShellState>((set) => ({
  primaryView: 'messages',
  contactsView: 'friends',
  activeConversationId: null,

  setPrimaryView: (primaryView) => set({ primaryView }),
  setContactsView: (contactsView) => set({ contactsView }),
  setActiveConversationId: (activeConversationId) => set({ activeConversationId }),
}))
