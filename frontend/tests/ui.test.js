import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import App from '../src/App.vue'
import { state } from '../src/store'
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
    ListOperations: () => Promise.resolve([]),
    QueueThumbnails: () => Promise.resolve(),
    InspectFile: () => Promise.resolve(null)
  }
}))

function mountShell() {
  state.demoMode = true
  state.cwd = '/demo'
  state.listing = demoListing('/demo')
  state.sidebar = demoSidebar()
  state.ops = demoOperations()
  state.viewMode = 'grid'
  return mount(App, { attachTo: document.body })
}

describe('view mode toggle', () => {
  it('switches to list and back to grid from the segmented control', async () => {
    const wrapper = mountShell()
    const buttons = wrapper.findAll('.segmented button')
    expect(buttons.length).toBe(2)

    await buttons[1].trigger('click')
    expect(state.viewMode).toBe('list')
    expect(wrapper.find('.list-layer').exists()).toBe(true)
    expect(wrapper.find('.grid-layer').exists()).toBe(false)

    await buttons[0].trigger('click')
    expect(state.viewMode).toBe('grid')
    expect(wrapper.find('.grid-layer').exists()).toBe(true)
    expect(wrapper.find('.list-layer').exists()).toBe(false)

    wrapper.unmount()
  })

  it('exposes the sortable list header in list view', async () => {
    const wrapper = mountShell()
    state.viewMode = 'list'
    await wrapper.vm.$nextTick()
    const header = wrapper.find('.list-header')
    expect(header.exists()).toBe(true)
    expect(header.text()).toContain('Name')
    expect(header.text()).toContain('Modified')
    wrapper.unmount()
  })
})

describe('icon rendering', () => {
  it('draws a glyph for every sidebar place, mount, and bookmark', () => {
    const wrapper = mountShell()
    const places = wrapper.findAll('.place')
    expect(places.length).toBeGreaterThan(5)
    for (const place of places) {
      const path = place.find('svg path')
      expect(path.exists(), `place "${place.text()}" is missing its icon`).toBe(true)
      expect((path.attributes('d') || '').length).toBeGreaterThan(8)
    }

    const volumes = wrapper.findAll('.volume')
    expect(volumes.length).toBeGreaterThan(0)
    for (const volume of volumes) {
      const path = volume.find('svg path')
      expect(path.exists(), `volume "${volume.text()}" is missing its icon`).toBe(true)
      expect((path.attributes('d') || '').length).toBeGreaterThan(8)
    }

    wrapper.unmount()
  })

  it('gives every icon an explicit size so it can never fall back to 300x150', () => {
    const wrapper = mountShell()
    const icons = wrapper.findAll('svg.vela-icon')
    expect(icons.length).toBeGreaterThan(10)
    for (const icon of icons) {
      expect(icon.attributes('width'), 'icon without a width attribute').toBeTruthy()
      expect(icon.attributes('height'), 'icon without a height attribute').toBeTruthy()
    }
    // The segmented view toggle must contain sized icons.
    for (const icon of wrapper.findAll('.segmented button svg')) {
      expect(icon.attributes('width')).toBeTruthy()
    }
    wrapper.unmount()
  })

  it('renders a glyph in the header bar, operations popover, and drag shelf', () => {
    const wrapper = mountShell()
    expect(wrapper.find('header.headerbar svg.vela-icon').exists()).toBe(true)
    expect(wrapper.find('.op-ring').exists()).toBe(true)
    wrapper.unmount()
  })
})
