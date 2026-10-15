import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'

import { formatSize, formatSpeed, formatEta, formatDate, relativeTime, iconPath, iconTint } from '../src/utils/format'

vi.mock('../src/api', () => ({
  usingWails: () => false,
  onEvent: () => () => {},
  initEvents: () => {},
  backend: {
    GetConfig: () => Promise.resolve(null),
    GetSidebar: () => Promise.resolve({ places: [], volumes: [], bookmarks: [] }),
    ListDirectory: () => Promise.resolve(null),
    SaveConfig: () => Promise.resolve({}),
    ListOperations: () => Promise.resolve([]),
    QueueThumbnails: () => Promise.resolve()
  },
  call: () => Promise.resolve(null)
}))

describe('formatters', () => {
  it('formats sizes with Linux decimal units', () => {
    expect(formatSize(0)).toBe('—')
    expect(formatSize(999)).toBe('999 B')
    expect(formatSize(1500)).toBe('1.5 kB')
    expect(formatSize(52_428_800)).toBe('52 MB')
    expect(formatSize(1_288_490_188)).toBe('1.3 GB')
  })

  it('formats transfer speed and ETA', () => {
    expect(formatSpeed(68_400_000)).toBe('68 MB/s')
    expect(formatEta(11)).toBe('11s left')
    expect(formatEta(150)).toBe('2m 30s left')
    expect(formatEta(0)).toBe('')
  })

  it('formats dates and relative times', () => {
    const now = Date.now()
    expect(formatDate(0)).toBe('—')
    expect(formatDate(now)).toMatch(/^Today, /)
    expect(relativeTime(now - 30_000)).toBe('just now')
    expect(relativeTime(now - 3 * 60 * 60 * 1000)).toBe('3 h ago')
  })

  it('picks icons and tints per entry category', () => {
    const folder = { isDir: true, category: 'folder', extension: '' }
    const image = { isDir: false, category: 'image', extension: 'jpg' }
    // The Go backend classifies source files as category "code".
    const code = { isDir: false, category: 'code', extension: 'go' }
    const archive = { isDir: false, category: 'archive', extension: 'zip' }
    expect(iconPath(folder)).toContain('M3 6.5')
    expect(iconTint(folder)).toBe('tint-folder')
    expect(iconTint(image)).toBe('tint-image')
    expect(iconTint(code)).toBe('tint-code')
    expect(iconTint(archive)).toBe('tint-archive')
    // Extension mapping wins for the icon glyph.
    expect(iconPath(code)).toBe(iconPath({ isDir: false, category: 'code', extension: 'ts' }))
  })
})
