import type { Editor } from '@tiptap/core'

export interface EditorAction {
  /** The sprite icon and the `editor.<name>` translation key. */
  name: string
  icon: string
  /** ProseMirror shortcut notation, spelled for the reader in the tooltip. */
  shortcut?: string
  run: (editor: Editor) => void
  /** Only the toggling actions carry one — the button then reports `aria-pressed`. */
  isActive?: (editor: Editor) => boolean
  /** Omitted where the action is always available. */
  isEnabled?: (editor: Editor) => boolean
}

// Reading order: history, marks, block types, lists, then the pair that
// inserts or clears. Each group is a divider in the toolbar.
export const ACTION_GROUPS: EditorAction[][] = [
  [
    {
      name: 'undo',
      icon: 'undo',
      shortcut: 'Mod-z',
      run: (editor) => editor.chain().focus().undo().run(),
      isEnabled: (editor) => editor.can().undo()
    },
    {
      name: 'redo',
      icon: 'redo',
      shortcut: 'Mod-Shift-z',
      run: (editor) => editor.chain().focus().redo().run(),
      isEnabled: (editor) => editor.can().redo()
    }
  ],
  [
    {
      name: 'bold',
      icon: 'bold',
      shortcut: 'Mod-b',
      run: (editor) => editor.chain().focus().toggleBold().run(),
      isActive: (editor) => editor.isActive('bold')
    },
    {
      name: 'italic',
      icon: 'italic',
      shortcut: 'Mod-i',
      run: (editor) => editor.chain().focus().toggleItalic().run(),
      isActive: (editor) => editor.isActive('italic')
    },
    {
      name: 'strike',
      icon: 'strike',
      shortcut: 'Mod-Shift-s',
      run: (editor) => editor.chain().focus().toggleStrike().run(),
      isActive: (editor) => editor.isActive('strike')
    }
  ],
  [
    {
      name: 'paragraph',
      icon: 'paragraph',
      shortcut: 'Mod-Alt-0',
      run: (editor) => editor.chain().focus().setParagraph().run(),
      isActive: (editor) => editor.isActive('paragraph')
    },
    {
      name: 'h1',
      icon: 'h1',
      shortcut: 'Mod-Alt-1',
      run: (editor) => editor.chain().focus().toggleHeading({ level: 1 }).run(),
      isActive: (editor) => editor.isActive('heading', { level: 1 })
    },
    {
      name: 'h2',
      icon: 'h2',
      shortcut: 'Mod-Alt-2',
      run: (editor) => editor.chain().focus().toggleHeading({ level: 2 }).run(),
      isActive: (editor) => editor.isActive('heading', { level: 2 })
    },
    {
      name: 'h3',
      icon: 'h3',
      shortcut: 'Mod-Alt-3',
      run: (editor) => editor.chain().focus().toggleHeading({ level: 3 }).run(),
      isActive: (editor) => editor.isActive('heading', { level: 3 })
    },
    {
      name: 'blockquote',
      icon: 'blockquote',
      shortcut: 'Mod-Shift-b',
      run: (editor) => editor.chain().focus().toggleBlockquote().run(),
      isActive: (editor) => editor.isActive('blockquote')
    }
  ],
  [
    {
      name: 'bulletlist',
      icon: 'bulletlist',
      shortcut: 'Mod-Shift-8',
      run: (editor) => editor.chain().focus().toggleBulletList().run(),
      isActive: (editor) => editor.isActive('bulletList')
    },
    {
      name: 'orderedList',
      icon: 'orderedlist',
      shortcut: 'Mod-Shift-7',
      run: (editor) => editor.chain().focus().toggleOrderedList().run(),
      isActive: (editor) => editor.isActive('orderedList')
    }
  ],
  [
    {
      name: 'horizontalRule',
      icon: 'minus',
      run: (editor) => editor.chain().focus().setHorizontalRule().run()
    },
    {
      name: 'clearFormatting',
      icon: 'eraser',
      run: (editor) => editor.chain().focus().unsetAllMarks().clearNodes().run()
    }
  ]
]

export const ACTIONS = ACTION_GROUPS.flat()

// TipTap spells a shortcut the ProseMirror way; a tooltip spells it the way
// this keyboard does, so the hint reads Mod-b as ⌘B on a Mac and Ctrl+B
// anywhere else.
const MODIFIER_LABELS: Record<string, [string, string]> = {
  Mod: ['Ctrl', '⌘'],
  Shift: ['Shift', '⇧'],
  Alt: ['Alt', '⌥']
}

export function shortcutLabel(shortcut: string, isMac: boolean): string {
  const parts = shortcut.split('-').map((part) => {
    const labels = MODIFIER_LABELS[part]
    if (labels) return isMac ? labels[1] : labels[0]
    return part.toUpperCase()
  })
  return parts.join(isMac ? '' : '+')
}
