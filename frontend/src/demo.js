/**
 * Demo dataset used when no backend is attached (pure browser preview).
 * Lets designers review the full UI — virtual list, marquee, inspector — with
 * realistic content while the Go backend is unavailable.
 */

const now = Date.now()
const day = 86400000

const folder = (name, path, count) => ({
  name, path, size: 0, modTimeUnix: now - day * 3, isDir: true, isSymlink: false,
  isHidden: false, isExecutable: false, extension: '', mimeType: 'inode/directory',
  category: 'folder', permissions: 'drwxr-xr-x', itemCount: count
})

const file = (name, ext, size, category, mime, daysAgo) => ({
  name, path: '/demo/' + name, size, modTimeUnix: now - day * daysAgo, isDir: false,
  isSymlink: false, isHidden: false, isExecutable: false, extension: ext,
  mimeType: mime, category, permissions: '-rw-r--r--'
})

export function demoListing(path) {
  const entries = [
    folder('Documents', '/demo/Documents', 42),
    folder('Downloads', '/demo/Downloads', 18),
    folder('Pictures', '/demo/Pictures', 1284),
    folder('Projects', '/demo/Projects', 9),
    folder('Videos', '/demo/Videos', 6),
    file('DSC_0142.ARW', 'arw', 52_428_800, 'image', 'image/x-sony-arw', 1),
    file('DSC_0143.ARW', 'arw', 54_001_664, 'image', 'image/x-sony-arw', 1),
    file('DSC_0144.JPG', 'jpg', 8_912_896, 'image', 'image/jpeg', 2),
    file('DSC_0145.JPG', 'jpg', 9_224_192, 'image', 'image/jpeg', 2),
    file('holiday-video.mp4', 'mp4', 1_288_490_188, 'video', 'video/mp4', 4),
    file('soundtrack.flac', 'flac', 48_234_496, 'audio', 'audio/flac', 12),
    file('quarterly-report.pdf', 'pdf', 2_411_724, 'pdf', 'application/pdf', 6),
    file('backup-2026-03.tar.gz', 'gz', 743_440_384, 'archive', 'application/x-archive', 9),
    file('notes.md', 'md', 12_288, 'document', 'text/plain', 0),
    file('main.go', 'go', 8_192, 'code', 'text/plain', 0),
    file('App.vue', 'vue', 24_576, 'code', 'text/plain', 0),
    file('invoice-2026-014.pdf', 'pdf', 184_320, 'pdf', 'application/pdf', 15),
    file('screenshot-2026-10-14.png', 'png', 1_048_576, 'image', 'image/png', 1),
    file('data-export.csv', 'csv', 4_194_304, 'document', 'text/plain', 7),
    file('LICENSE', '', 35_000, 'document', 'text/plain', 300),
    file('install.sh', 'sh', 2_048, 'code', 'text/plain', 22),
    file('.env.local', '', 512, 'document', 'text/plain', 3),
    file('.gitignore', '', 256, 'document', 'text/plain', 40)
  ].map((e) => ({ ...e, path: e.path.replace('/demo/', (path && path !== 'trash://' ? path : '/demo') + '/') }))

  return {
    path: path || '/demo',
    parentPath: '/',
    breadcrumbs: [
      { name: '/', path: '/' },
      { name: 'home', path: '/home' },
      { name: 'user', path: '/home/user' },
      { name: (path || '/demo').split('/').filter(Boolean).pop() || 'demo', path: path || '/demo' }
    ],
    entries,
    totalCount: entries.length,
    hiddenCount: 2,
    totalSize: entries.reduce((sum, e) => sum + (e.size || 0), 0),
    diskTotal: 512_110_190_592,
    diskFree: 183_500_000_000,
    diskUsedPercent: 64.2,
    mountPoint: '/',
    fileSystemType: 'ext4',
    writable: true,
    isTrashView: path === 'trash://'
  }
}

export function demoSidebar() {
  return {
    homeDir: '/home/user',
    userName: 'user',
    hostName: 'workstation',
    places: [
      { id: 'home', name: 'Home', path: '/home/user', icon: 'home', kind: 'xdg' },
      { id: 'desktop', name: 'Desktop', path: '/home/user/Desktop', icon: 'desktop', kind: 'xdg' },
      { id: 'documents', name: 'Documents', path: '/home/user/Documents', icon: 'documents', kind: 'xdg' },
      { id: 'downloads', name: 'Downloads', path: '/home/user/Downloads', icon: 'downloads', kind: 'xdg' },
      { id: 'music', name: 'Music', path: '/home/user/Music', icon: 'music', kind: 'xdg' },
      { id: 'pictures', name: 'Pictures', path: '/home/user/Pictures', icon: 'pictures', kind: 'xdg' },
      { id: 'videos', name: 'Videos', path: '/home/user/Videos', icon: 'videos', kind: 'xdg' },
      { id: 'trash', name: 'Trash', path: 'trash://', icon: 'trash', kind: 'trash', badge: 7 }
    ],
    volumes: [
      {
        name: 'System Root', mountPoint: '/', device: '/dev/nvme0n1p2', fsType: 'ext4',
        totalBytes: 512_110_190_592, freeBytes: 183_500_000_000, usedBytes: 328_610_190_592,
        usedPercent: 64.2, isRemovable: false, hasTrash: false, trashCount: 0
      },
      {
        name: 'SanDisk USB', mountPoint: '/media/user/SanDisk', device: '/dev/sdb1', fsType: 'exfat',
        totalBytes: 256_060_514_304, freeBytes: 158_000_000_000, usedBytes: 98_060_514_304,
        usedPercent: 38.3, isRemovable: true, hasTrash: true, trashCount: 3
      }
    ],
    bookmarks: [
      { id: 'bm-0', name: 'workspace', path: '/workspace', icon: 'bookmark', kind: 'bookmark' },
      { id: 'bm-1', name: 'config', path: '/home/user/.config', icon: 'bookmark', kind: 'bookmark' }
    ]
  }
}

export function demoOperations() {
  return [
    {
      id: 'copy-demo', kind: 'copy', state: 'running', destDir: '/media/user/SanDisk/Backup',
      sources: ['/demo/DSC_0142.ARW', '/demo/DSC_0143.ARW'], totalBytes: 1_900_000_000,
      doneBytes: 1_216_000_000, progress: 64, speedBps: 68_400_000, etaSeconds: 11,
      currentFile: '/demo/DSC_0142.ARW', itemsDone: 14, itemsTotal: 24,
      startedAt: new Date().toISOString(), canUndo: false, conflictMode: 'rename'
    }
  ]
}

export function demoConfig() {
  return {
    theme: 'system', viewMode: 'grid', iconSize: 112, sortBy: 'name', sortDesc: false,
    showHidden: false, splitView: false, conflictMode: 'rename',
    lastPath: '/demo', bookmarks: ['/workspace', '/home/user/.config'], sidebarWidth: 232
  }
}
