import { describe, expect, it, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'

import App from '../src/App.vue'
import { state, activeCwd, activeSelection } from '../src/store'
import { demoListing, demoSidebar } from '../src/demo'
import { targetPaths } from '../src/storeFiles'
import { toggleSplitView, navigate, selectOnly, setActivePane } from '../src/storeNav'
import { transferToOtherPane } from '../src/storeFiles2'

const calls = []

vi.mock('../src/api', () => ({
  usingWails: () => false,
  onEvent: () => () => {},
  initEvents: () => {},
  call: (method, ...args) => {
    calls.push({ method, args })
    return Promise.resolve(null)
  },
  backend: {
    GetConfig: () => Promise.reject(new Error('offline')),
    GetSidebar: () => Promise.resolve(demoSidebar()),
    ListDirectory: (path) => Promise.resolve(demoListing(path)),
    SaveConfig: () => Promise.resolve({}),
    ListOperations: () => Promise.resolve([]),
    QueueThumbnails: () => Promise.resolve(),
    InspectFile: () => Promise.resolve(null),
    Copy: (sources, dest, mode) => {
      calls.push({ method: 'Copy', args: [sources, dest, mode] })
      return Promise.resolve('copy-1')
    },
    Move: (sources, dest, mode) => {
      calls.push({ method: 'Move', args: [sources, dest, mode] })
      return Promise.resolve('move-1')
    }
  }
}))

async function mountShell() {
  state.demoMode = true
  state.cwd = '/demo'
  state.listing = demoListing('/demo')
  state.sidebar = demoSidebar()
  state.ops = []
  state.selection = []
  state.splitView = false
  state.activePane = 'primary'
  state.secondary.cwd = ''
  state.secondary.entries = []
  state.secondary.selection = []
  return mount(App, { attachTo: document.body })
}

describe('dual pane', () => {
  beforeEach(() => {
    calls.length = 0
  })

  it('opens an independent second pane when split view is enabled', async () => {
    const wrapper = await mountShell()
    expect(wrapper.findAll('.pane').length).toBe(1)

    await toggleSplitView()
    await wrapper.vm.$nextTick()

    const panes = wrapper.findAll('.pane')
    expect(panes.length).toBe(2)
    expect(panes[1].classes()).toContain('pane-secondary')
    expect(state.secondary.cwd).toBeTruthy()
    expect(state.secondary.cwd).not.toBe(state.cwd)
    expect(state.secondary.entries.length).toBeGreaterThan(0)

    wrapper.unmount()
  })

  it('keeps pane selections separate and routes actions to the active pane', async () => {
    const wrapper = await mountShell()
    await toggleSplitView()
    await wrapper.vm.$nextTick()

    const primaryPath = state.listing.entries[0].path
    const secondaryPath = state.secondary.entries[1].path

    setActivePane('primary')
    selectOnly(primaryPath, 0, 'primary')
    setActivePane('secondary')
    selectOnly(secondaryPath, 1, 'secondary')

    expect(state.selection).toEqual([primaryPath])
    expect(state.secondary.selection).toEqual([secondaryPath])
    expect(activeCwd()).toBe(state.secondary.cwd)
    expect(activeSelection()).toEqual([secondaryPath])
    expect(targetPaths()).toEqual([secondaryPath])

    setActivePane('primary')
    expect(targetPaths()).toEqual([primaryPath])
    expect(activeCwd()).toBe(state.cwd)

    wrapper.unmount()
  })

  it('highlights the active pane', async () => {
    const wrapper = await mountShell()
    await toggleSplitView()
    await wrapper.vm.$nextTick()

    setActivePane('primary')
    await wrapper.vm.$nextTick()
    let panes = wrapper.findAll('.pane')
    expect(panes[0].classes()).toContain('pane-active')
    expect(panes[1].classes()).not.toContain('pane-active')

    setActivePane('secondary')
    await wrapper.vm.$nextTick()
    panes = wrapper.findAll('.pane')
    expect(panes[1].classes()).toContain('pane-active')

    wrapper.unmount()
  })

  it('copies the active selection into the other pane', async () => {
    const wrapper = await mountShell()
    await toggleSplitView()
    await wrapper.vm.$nextTick()

    setActivePane('primary')
    const source = state.listing.entries.find((e) => !e.isDir)
    selectOnly(source.path, 0, 'primary')

    await transferToOtherPane(false)
    const copyCall = calls.find((c) => c.method === 'Copy')
    expect(copyCall).toBeTruthy()
    expect(copyCall.args[0]).toEqual([source.path])
    expect(copyCall.args[1]).toBe(state.secondary.cwd)

    // Moving from the second pane targets the primary folder.
    setActivePane('secondary')
    const secondarySource = state.secondary.entries.find((e) => !e.isDir)
    selectOnly(secondarySource.path, 0, 'secondary')
    await transferToOtherPane(true)
    const moveCall = calls.find((c) => c.method === 'Move')
    expect(moveCall).toBeTruthy()
    expect(moveCall.args[0]).toEqual([secondarySource.path])
    expect(moveCall.args[1]).toBe(state.cwd)

    wrapper.unmount()
  })

  it('does not change the second pane when the first navigates', async () => {
    const wrapper = await mountShell()
    await toggleSplitView()
    await wrapper.vm.$nextTick()
    const secondaryBefore = state.secondary.cwd

    const folder = state.listing.entries.find((e) => e.isDir)
    await navigate(folder.path)
    await wrapper.vm.$nextTick()

    expect(state.cwd).toBe(folder.path)
    expect(state.secondary.cwd).toBe(secondaryBefore)

    wrapper.unmount()
  })
})
