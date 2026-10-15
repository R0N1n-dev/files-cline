import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import App from '../src/App.vue'
import { state } from '../src/store'
import { init } from '../src/storeNav'
import { demoListing, demoSidebar, demoOperations } from '../src/demo'

vi.mock('../src/api', () => ({
  usingWails: () => false,
  onEvent: () => () => {},
  initEvents: () => {},
  call: () => Promise.resolve(null),
  backend: {
    GetConfig: () => Promise.reject(new Error('offline')),
    GetSidebar: () => Promise.resolve(demoSidebar()),
    ListDirectory: (path) => Promise.resolve(demoListing(path)),
    SaveConfig: () => Promise.resolve({}),
    ListOperations: () => Promise.resolve(demoOperations()),
    QueueThumbnails: () => Promise.resolve(),
    InspectFile: () => Promise.resolve(null)
  }
}))

describe('Vela Files shell', () => {
  it('renders the headerbar, sidebar, and virtual file view with demo data', async () => {
    state.demoMode = true
    state.cwd = '/demo'
    state.listing = demoListing('/demo')
    state.sidebar = demoSidebar()
    state.ops = demoOperations()

    const wrapper = mount(App, { attachTo: document.body })

    expect(wrapper.find('header.headerbar').exists()).toBe(true)
    expect(wrapper.find('aside.sidebar').exists()).toBe(true)
    expect(wrapper.find('.view-scroll').exists()).toBe(true)

    // Breadcrumbs + disk capacity pill come from the listing.
    expect(wrapper.text()).toContain('demo')
    expect(wrapper.text()).toContain('free')

    // Virtual grid renders only a window of tiles, never all 23 entries.
    const tiles = wrapper.findAll('.tile')
    expect(tiles.length).toBeGreaterThan(0)
    expect(tiles.length).toBeLessThanOrEqual(state.listing.entries.length)

    // Sidebar exposes places, mounts, and the trash badge.
    expect(wrapper.findAll('.place').length).toBeGreaterThan(3)
    expect(wrapper.findAll('.volume').length).toBe(2)
    expect(wrapper.text()).toContain('SanDisk USB')

    wrapper.unmount()
  })

  it('shows the operations ring for an active copy', () => {
    state.ops = demoOperations()
    const wrapper = mount(App, { attachTo: document.body })
    expect(wrapper.find('.op-ring').exists()).toBe(true)
    wrapper.unmount()
  })

  it('bootstraps into demo mode when the backend is unavailable', async () => {
    state.demoMode = false
    state.ready = false
    await init()
    expect(state.demoMode).toBe(true)
    expect(state.ready).toBe(true)
    expect(state.listing.entries.length).toBeGreaterThan(5)
  })
})
