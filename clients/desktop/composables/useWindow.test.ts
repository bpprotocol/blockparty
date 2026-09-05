import { describe, expect, it } from 'vitest'
import { needsCustomControls, reservesTrafficLightSpace } from './useWindow'

describe('window chrome per platform', () => {
  it('draws its own buttons on Windows and Linux', () => {
    expect(needsCustomControls('win32')).toBe(true)
    expect(needsCustomControls('linux')).toBe(true)
  })

  it('leaves macOS to its traffic lights, and reserves room for them', () => {
    expect(needsCustomControls('darwin')).toBe(false)
    expect(reservesTrafficLightSpace('darwin')).toBe(true)
    expect(reservesTrafficLightSpace('linux')).toBe(false)
  })

  it('draws nothing outside Electron, where the browser has its own chrome', () => {
    expect(needsCustomControls(undefined)).toBe(false)
    expect(reservesTrafficLightSpace(undefined)).toBe(false)
  })
})
