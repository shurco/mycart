import { describe, it, expect, vi, beforeEach } from 'vitest'
import { loadData, saveData, deleteData, toggleActive } from './apiHelpers'
import * as api from './index'

// Mock the API module
vi.mock('./index', () => ({
  apiGet: vi.fn(),
  apiPost: vi.fn(),
  apiUpdate: vi.fn(),
  apiDelete: vi.fn(),
  showMessage: vi.fn()
}))

describe('API Helpers', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('loadData', () => {
    it('should successfully load data', async () => {
      const mockData = { id: '1', name: 'Test Product' }
      vi.mocked(api.apiGet).mockResolvedValue({
        success: true,
        result: mockData,
        message: 'Success'
      })

      const result = await loadData('/api/products')

      expect(result).toEqual(mockData)
      expect(api.apiGet).toHaveBeenCalledWith('/api/products')
    })

    it('should return null on API failure', async () => {
      vi.mocked(api.apiGet).mockResolvedValue({
        success: false,
        message: 'Not found'
      })

      const result = await loadData('/api/products')

      expect(result).toBeNull()
      expect(api.showMessage).toHaveBeenCalledWith('Not found', 'connextError')
    })

    it('should handle network errors', async () => {
      vi.mocked(api.apiGet).mockRejectedValue(new Error('Network error'))

      const result = await loadData('/api/products', 'Custom error message')

      expect(result).toBeNull()
      expect(api.showMessage).toHaveBeenCalledWith('Custom error message', 'connextError')
    })
  })

  describe('saveData', () => {
    it('should save new data using POST', async () => {
      const mockData = { id: '1', name: 'New Product' }
      const postData = { name: 'New Product' }

      vi.mocked(api.apiPost).mockResolvedValue({
        success: true,
        result: mockData,
        message: 'Created'
      })

      const result = await saveData('/api/products', postData, false, 'Product created')

      expect(result).toEqual(mockData)
      expect(api.apiPost).toHaveBeenCalledWith('/api/products', postData)
      expect(api.showMessage).toHaveBeenCalledWith('Product created', 'connextSuccess')
    })

    it('should update existing data using PUT', async () => {
      const mockData = { id: '1', name: 'Updated Product' }
      const updateData = { name: 'Updated Product' }

      vi.mocked(api.apiUpdate).mockResolvedValue({
        success: true,
        result: mockData,
        message: 'Updated'
      })

      const result = await saveData('/api/products/1', updateData, true, 'Product updated')

      expect(result).toEqual(mockData)
      expect(api.apiUpdate).toHaveBeenCalledWith('/api/products/1', updateData)
      expect(api.showMessage).toHaveBeenCalledWith('Product updated', 'connextSuccess')
    })

    it('should return null on save failure', async () => {
      vi.mocked(api.apiPost).mockResolvedValue({
        success: false,
        message: 'Validation error'
      })

      const result = await saveData('/api/products', {}, false)

      expect(result).toBeNull()
      expect(api.showMessage).toHaveBeenCalledWith('Validation error', 'connextError')
    })

    it('should use default messages', async () => {
      const mockData = { id: '1', name: 'Product' }

      vi.mocked(api.apiPost).mockResolvedValue({
        success: true,
        result: mockData,
        message: 'Success'
      })

      await saveData('/api/products', {}, false)

      expect(api.showMessage).toHaveBeenCalledWith('Data saved', 'connextSuccess')
    })
  })

  describe('deleteData', () => {
    it('should successfully delete data', async () => {
      // deleteData returns true only if result is truthy
      vi.mocked(api.apiDelete).mockResolvedValue({
        success: true,
        result: { deleted: true },  // Return truthy result
        message: 'Deleted'
      })

      const result = await deleteData('/api/products/1')

      expect(result).toBe(true)
      expect(api.apiDelete).toHaveBeenCalledWith('/api/products/1')
      expect(api.showMessage).toHaveBeenCalledWith('Deleted successfully', 'connextSuccess')
    })

    it('should return false on delete failure', async () => {
      vi.mocked(api.apiDelete).mockResolvedValue({
        success: false,
        message: 'Cannot delete'
      })

      const result = await deleteData('/api/products/1')

      expect(result).toBe(false)
      expect(api.showMessage).toHaveBeenCalledWith('Cannot delete', 'connextError')
    })

    it('should handle network errors', async () => {
      vi.mocked(api.apiDelete).mockRejectedValue(new Error('Network error'))

      const result = await deleteData('/api/products/1', 'Deleted', 'Custom error')

      expect(result).toBe(false)
      expect(api.showMessage).toHaveBeenCalledWith('Custom error', 'connextError')
    })
  })

  describe('toggleActive', () => {
    it('should successfully toggle active status', async () => {
      const mockData = { id: '1', active: true }

      vi.mocked(api.apiUpdate).mockResolvedValue({
        success: true,
        result: mockData,
        message: 'Updated'
      })

      const result = await toggleActive('/api/products/1/toggle')

      expect(result).toEqual(mockData)
      expect(api.apiUpdate).toHaveBeenCalledWith('/api/products/1/toggle', {})
      expect(api.showMessage).toHaveBeenCalledWith('Status updated', 'connextSuccess')
    })

    it('should return null on toggle failure', async () => {
      vi.mocked(api.apiUpdate).mockResolvedValue({
        success: false,
        message: 'Update failed'
      })

      const result = await toggleActive('/api/products/1/toggle')

      expect(result).toBeNull()
      expect(api.showMessage).toHaveBeenCalledWith('Update failed', 'connextError')
    })

    it('should handle network errors', async () => {
      vi.mocked(api.apiUpdate).mockRejectedValue(new Error('Network error'))

      const result = await toggleActive('/api/products/1/toggle', 'Toggled', 'Custom error')

      expect(result).toBeNull()
      expect(api.showMessage).toHaveBeenCalledWith('Custom error', 'connextError')
    })

    it('should use custom success message', async () => {
      const mockData = { id: '1', active: false }

      vi.mocked(api.apiUpdate).mockResolvedValue({
        success: true,
        result: mockData,
        message: 'Success'
      })

      await toggleActive('/api/products/1/toggle', 'Product activated')

      expect(api.showMessage).toHaveBeenCalledWith('Product activated', 'connextSuccess')
    })
  })
})
