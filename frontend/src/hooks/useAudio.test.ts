// @vitest-environment happy-dom
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { renderHook, act } from '@testing-library/react'
import { useAudio } from './useAudio'

// ---------------------------------------------------------------------------
// HTMLAudioElement mock
// ---------------------------------------------------------------------------
class MockAudio {
  src = ''
  preload = ''
  onended: (() => void) | null = null
  onerror: (() => void) | null = null

  load = vi.fn()
  pause = vi.fn()
  play = vi.fn(() => Promise.resolve())

  static instances: MockAudio[] = []

  constructor(src?: string) {
    this.src = src ?? ''
    MockAudio.instances.push(this)
  }

  static reset() {
    MockAudio.instances = []
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
let revokeObjectURL: any

beforeEach(() => {
  MockAudio.reset()
  vi.stubGlobal('Audio', MockAudio)
  vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:mock-url')
  revokeObjectURL = vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

// ---------------------------------------------------------------------------
// preload
// ---------------------------------------------------------------------------
describe('preload', () => {
  it('creates an Audio element and calls load()', () => {
    const { result } = renderHook(() => useAudio())
    act(() => { result.current.preload('http://example.com/test.ogg') })
    expect(MockAudio.instances).toHaveLength(1)
    expect(MockAudio.instances[0].load).toHaveBeenCalledOnce()
  })

  it('does nothing when called twice with the same URL', () => {
    const { result } = renderHook(() => useAudio())
    act(() => {
      result.current.preload('http://example.com/a.ogg')
      result.current.preload('http://example.com/a.ogg')
    })
    expect(MockAudio.instances).toHaveLength(1)
  })

  it('replaces preloaded element when URL changes', () => {
    const { result } = renderHook(() => useAudio())
    act(() => { result.current.preload('http://example.com/a.ogg') })
    act(() => { result.current.preload('http://example.com/b.ogg') })
    expect(MockAudio.instances).toHaveLength(2)
  })

  it('does nothing for an empty URL', () => {
    const { result } = renderHook(() => useAudio())
    act(() => { result.current.preload('') })
    expect(MockAudio.instances).toHaveLength(0)
  })
})

// ---------------------------------------------------------------------------
// playUrl
// ---------------------------------------------------------------------------
describe('playUrl', () => {
  it('calls play() and sets isPlaying when play resolves', async () => {
    const { result } = renderHook(() => useAudio())
    await act(async () => { result.current.playUrl('http://example.com/test.ogg') })
    expect(result.current.isPlaying).toBe(true)
    expect(MockAudio.instances[0].play).toHaveBeenCalledOnce()
  })

  it('sets isError=true and isPlaying=false when play rejects', async () => {
    const { result } = renderHook(() => useAudio())
    vi.stubGlobal('Audio', class extends MockAudio {
      play = vi.fn(() => Promise.reject(new Error('blocked')))
    })
    await act(async () => { result.current.playUrl('http://example.com/test.ogg') })
    expect(result.current.isPlaying).toBe(false)
    expect(result.current.isError).toBe(true)
  })

  it('reuses the preloaded element when URL matches', async () => {
    const { result } = renderHook(() => useAudio())
    act(() => { result.current.preload('http://example.com/test.ogg') })
    await act(async () => { result.current.playUrl('http://example.com/test.ogg') })
    // One element created by preload, reused by playUrl — not two
    expect(MockAudio.instances).toHaveLength(1)
  })

  it('creates a fresh element when URL differs from preloaded', async () => {
    const { result } = renderHook(() => useAudio())
    act(() => { result.current.preload('http://example.com/a.ogg') })
    await act(async () => { result.current.playUrl('http://example.com/b.ogg') })
    expect(MockAudio.instances).toHaveLength(2)
  })

  it('stops previous audio before playing new one', async () => {
    const { result } = renderHook(() => useAudio())
    await act(async () => { result.current.playUrl('http://example.com/a.ogg') })
    const first = MockAudio.instances[0]
    await act(async () => { result.current.playUrl('http://example.com/b.ogg') })
    expect(first.pause).toHaveBeenCalled()
  })

  it('clears isPlaying when onended fires', async () => {
    const { result } = renderHook(() => useAudio())
    await act(async () => { result.current.playUrl('http://example.com/test.ogg') })
    expect(result.current.isPlaying).toBe(true)
    act(() => { MockAudio.instances[0].onended?.() })
    expect(result.current.isPlaying).toBe(false)
  })

  it('sets isError=true when onerror fires', async () => {
    const { result } = renderHook(() => useAudio())
    await act(async () => { result.current.playUrl('http://example.com/test.ogg') })
    act(() => { MockAudio.instances[0].onerror?.() })
    expect(result.current.isError).toBe(true)
    expect(result.current.isPlaying).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// playBlob
// ---------------------------------------------------------------------------
describe('playBlob', () => {
  it('creates an object URL and calls play()', async () => {
    const { result } = renderHook(() => useAudio())
    const blob = new Blob([new Uint8Array([0])], { type: 'audio/ogg' })
    await act(async () => { result.current.playBlob(blob) })
    expect(MockAudio.instances[0].src).toBe('blob:mock-url')
    expect(MockAudio.instances[0].play).toHaveBeenCalledOnce()
    expect(result.current.isPlaying).toBe(true)
  })

  it('revokes objectURL in onended, not synchronously', async () => {
    const { result } = renderHook(() => useAudio())
    const blob = new Blob([new Uint8Array([0])], { type: 'audio/ogg' })
    await act(async () => { result.current.playBlob(blob) })
    expect(revokeObjectURL).not.toHaveBeenCalled()
    act(() => { MockAudio.instances[0].onended?.() })
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:mock-url')
  })

  it('revokes objectURL in onerror', async () => {
    const { result } = renderHook(() => useAudio())
    const blob = new Blob([new Uint8Array([0])], { type: 'audio/ogg' })
    await act(async () => { result.current.playBlob(blob) })
    act(() => { MockAudio.instances[0].onerror?.() })
    expect(revokeObjectURL).toHaveBeenCalledWith('blob:mock-url')
  })
})

// ---------------------------------------------------------------------------
// stop
// ---------------------------------------------------------------------------
describe('stop', () => {
  it('pauses audio and clears isPlaying', async () => {
    const { result } = renderHook(() => useAudio())
    await act(async () => { result.current.playUrl('http://example.com/test.ogg') })
    expect(result.current.isPlaying).toBe(true)
    act(() => { result.current.stop() })
    expect(MockAudio.instances[0].pause).toHaveBeenCalled()
    expect(result.current.isPlaying).toBe(false)
  })
})

// ---------------------------------------------------------------------------
// unmount cleanup
// ---------------------------------------------------------------------------
describe('unmount cleanup', () => {
  it('pauses audio and clears src on unmount', async () => {
    const { result, unmount } = renderHook(() => useAudio())
    await act(async () => { result.current.playUrl('http://example.com/test.ogg') })
    const audio = MockAudio.instances[0]
    unmount()
    expect(audio.pause).toHaveBeenCalled()
    expect(audio.src).toBe('')
  })
})
