import { defineStore } from 'pinia'
import { reactive, computed } from 'vue'

export const usePresenceStore = defineStore('presence', () => {
  // convUUID -> array of viewer objects {user_id, first_name, avatar_url}
  const viewers = reactive(new Map())

  function updatePresence(convUUID, viewerList) {
    if (viewerList.length === 0) {
      viewers.delete(convUUID)
    } else {
      viewers.set(convUUID, viewerList)
    }
  }

  function getViewers(convUUID, excludeUserID) {
    const list = viewers.get(convUUID) || []
    if (!excludeUserID) return list
    return list.filter(v => v.user_id !== excludeUserID)
  }

  function getViewerCount(convUUID, excludeUserID) {
    return getViewers(convUUID, excludeUserID).length
  }

  return {
    viewers,
    updatePresence,
    getViewers,
    getViewerCount
  }
})
