import { create } from 'zustand'

interface MessageComposerState {
  drafts: Record<number, string>
  setDraft: (conversationId: number, value: string) => void
}

export const useMessageComposerStore = create<MessageComposerState>((set) => ({
  drafts: {},
  setDraft: (conversationId, value) =>
    set((state) => ({
      drafts: {
        ...state.drafts,
        [conversationId]: value,
      },
    })),
}))
