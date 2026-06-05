import { useEffect, useRef, useState } from 'react'

export interface UseAudioReturn {
  isPlaying: boolean
  isError: boolean
  preload: (url: string) => void
  playUrl: (url: string) => void
  playBlob: (blob: Blob) => void
  stop: () => void
}

export function useAudio(): UseAudioReturn {
  const audioRef = useRef<HTMLAudioElement | null>(null)
  const preloadedUrlRef = useRef<string | null>(null)
  const [isPlaying, setIsPlaying] = useState(false)
  const [isError, setIsError] = useState(false)

  // Release media resource on unmount — prevents memory leak
  useEffect(() => {
    return () => {
      if (audioRef.current) {
        audioRef.current.pause()
        audioRef.current.src = ''
        audioRef.current = null
      }
    }
  }, [])

  function stopCurrent() {
    if (audioRef.current) {
      audioRef.current.pause()
      audioRef.current.onended = null
      audioRef.current.onerror = null
    }
    setIsPlaying(false)
    setIsError(false)
  }

  function preload(url: string) {
    if (!url || preloadedUrlRef.current === url) return
    stopCurrent()
    const audio = new Audio(url)
    audio.preload = 'auto'
    audio.load()
    audioRef.current = audio
    preloadedUrlRef.current = url
  }

  function playUrl(url: string) {
    setIsError(false)
    // Reuse preloaded element if URL matches; otherwise create fresh
    if (preloadedUrlRef.current !== url || !audioRef.current) {
      stopCurrent()
      audioRef.current = new Audio(url)
      preloadedUrlRef.current = url
    }
    const audio = audioRef.current
    audio.onended = () => setIsPlaying(false)
    audio.onerror = () => { setIsPlaying(false); setIsError(true) }
    audio.play().then(() => setIsPlaying(true)).catch(() => setIsError(true))
  }

  function playBlob(blob: Blob) {
    stopCurrent()
    setIsError(false)
    const url = URL.createObjectURL(blob)
    const audio = new Audio(url)
    // Revoke in onended/onerror, NOT synchronously — audio must finish loading first
    audio.onended = () => { setIsPlaying(false); URL.revokeObjectURL(url) }
    audio.onerror = () => { setIsPlaying(false); setIsError(true); URL.revokeObjectURL(url) }
    audioRef.current = audio
    preloadedUrlRef.current = null
    audio.play().then(() => setIsPlaying(true)).catch(() => setIsError(true))
  }

  return { isPlaying, isError, preload, playUrl, playBlob, stop: stopCurrent }
}
