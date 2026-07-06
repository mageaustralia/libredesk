export default class MessageCache {
    // Not reactive - see widget/store/chat.js for the reactive wrapper pattern.
    constructor(maxConvs = 30) {
        this.cache = new Map()
        this.maxConvs = maxConvs
        this.recentConvs = []
    }

    addMessages (convId, messages, page, totalPages) {
        const conv = this.cache.get(convId)
        const uniqueMsgs = messages.filter(m => !this.hasMessage(convId, m.uuid))

        if (conv) {
            conv.lastFetchedPage = Math.max(page, conv.lastFetchedPage)
            conv.hasMore = totalPages > conv.lastFetchedPage
            conv.totalPages = totalPages
            conv.pages.set(page, uniqueMsgs)
        } else {
            this.cache.set(convId, {
                pages: new Map([[page, uniqueMsgs]]),
                totalPages,
                lastFetchedPage: page,
                hasMore: totalPages > page,
            })
            this.pruneOldConversations(convId)
        }
    }

    purgeConversation (convId) {
        return this.cache.delete(convId)
    }

    hasMessage (convId, msgId) {
        return this._allMessages(convId).some(m => m.uuid === msgId)
    }

    addMessage (convId, message) {
        const conv = this.cache.get(convId)
        if (!conv || this.hasMessage(convId, message.uuid)) return
        if (!conv.pages.has(1)) {
            conv.pages.set(1, [message])
        } else {
            conv.pages.get(1).push(message)
        }
    }

    getAllPagesMessages (convId) {
        return this._allMessages(convId)
            .sort((a, b) => new Date(a.created_at) - new Date(b.created_at))
    }

    /**
     * Returns latest message for a conversation
     * @param {string} convId - Conversation ID
     * @param {string[]} type - Array of message types to filter - outgoing, incoming, etc.
     * @param {boolean} excludePrivate - Exclude private messages
     * @param {?function} excludeFn - Optional predicate; messages for which it
     *   returns true are excluded (e.g. DSN bounces, so they don't become the
     *   "latest" message used for recipient derivation).
     *
     * @returns {object} - Latest message object or null if not found
     */
    getLatestMessage (convId, type = [], excludePrivate = false, excludeFn = null) {
        const conv = this.cache.get(convId)
        if (!conv) return null

        // Get all messages from all pages
        let allMessages = Array.from(conv.pages.values()).flat()

        // Apply filters
        if (type.length > 0) {
            allMessages = allMessages.filter(msg => type.includes(msg.type))
        }
        if (excludePrivate) {
            allMessages = allMessages.filter(msg => !msg.private)
        }
        if (excludeFn) {
            allMessages = allMessages.filter(msg => !excludeFn(msg))
        }

        // Sort messages by created_at in descending order (newest first)
        allMessages.sort((a, b) => new Date(b.created_at) - new Date(a.created_at))

        return allMessages.length ? allMessages[0] : null
    }

    updateMessage (convId, msgId, updates) {
        const conv = this.cache.get(convId)
        if (!conv) return
        conv.pages.forEach(msgs => {
            const idx = msgs.findIndex(m => m.uuid === msgId)
            // Replace the message object instead of mutating in place. The
            // cache holds plain JS objects (not Vue reactive proxies), so
            // an in-place Object.assign is invisible to <MessageBubble>'s
            // computed reads of props.message.content. Swapping the array
            // entry produces a new object identity, which the parent's
            // v-for re-projects as a fresh prop.
            if (idx !== -1) msgs[idx] = { ...msgs[idx], ...updates }
        })
    }

    updateMessageField (convId, msgId, field, value) {
        this._updateMessageBy(convId, msgId, msg => { msg[field] = value })
    }

    removeMessage (convId, msgId) {
        const conv = this.cache.get(convId)
        if (!conv) return
        conv.pages.forEach(msgs => {
            const msgIndex = msgs.findIndex(m => m.uuid === msgId)
            if (msgIndex !== -1) {
                msgs.splice(msgIndex, 1)
            }
        })
    }

    hasMore (convId) {
        return this.cache.get(convId)?.hasMore || false
    }

    getLastFetchedPage (convId) {
        return this.cache.get(convId)?.lastFetchedPage || 0
    }

    pruneOldConversations (convId) {
        this.recentConvs = [convId, ...this.recentConvs.filter(id => id !== convId)]
        if (this.recentConvs.length > this.maxConvs) {
            const removed = this.recentConvs.pop()
            this.cache.delete(removed)
        }
    }

    hasConversation (convId) {
        return this.cache.has(convId)
    }

    _allMessages (convId) {
        const conv = this.cache.get(convId)
        if (!conv) return []
        return Array.from(conv.pages.values()).flat()
    }

    _updateMessageBy (convId, msgId, mutate) {
        const conv = this.cache.get(convId)
        if (!conv) return
        conv.pages.forEach(msgs => {
            const msg = msgs.find(m => m.uuid === msgId)
            if (msg) mutate(msg)
        })
    }
}
