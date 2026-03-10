// Strip +conv-{uuid-v4} from email if present.
// Only matches strict UUID v4 format (36 chars)
// e.g., support+conv-13216cf7-6626-4b0d-a938-46ce65a20701@domain.com -> support@domain.com
export function stripConvUUID (email) {
    if (!email) return email
    return email.replace(/\+conv-[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[a-f0-9]{4}-[a-f0-9]{12}@/i, '@')
}

export function computeRecipientsFromMessage (message, contactEmail, inboxEmail) {
    const meta = message?.meta || {}
    const isIncoming = message.type === 'incoming'
    const contactLower = (contactEmail || '').toLowerCase()

    // Build TO field — the conversation contact is always the primary recipient.
    let toList
    if (isIncoming) {
        if (meta.from && meta.from.length) {
            // Check if the contact email matches any of the from addresses.
            const fromLower = meta.from.map(e => e.toLowerCase())
            if (contactLower && !fromLower.includes(contactLower)) {
                // Contact was changed — use the new contact as To.
                toList = [contactEmail]
            } else {
                toList = meta.from
            }
        } else {
            toList = contactEmail ? [contactEmail] : []
        }
    } else {
        if (meta.to && meta.to.length) {
            // For outgoing, check if contact email is in the To list.
            const toLower = meta.to.map(e => e.toLowerCase())
            if (contactLower && !toLower.includes(contactLower)) {
                // Contact was changed — use the new contact as To.
                toList = [contactEmail]
            } else {
                toList = meta.to
            }
        } else {
            toList = contactEmail ? [contactEmail] : []
        }
    }

    // Build CC field
    let ccList = [...(meta.cc || [])]

    if (isIncoming) {
        // Include original 'to' recipients in CC to preserve full thread context.
        if (Array.isArray(meta.to))
            ccList = ccList.concat(meta.to)
    }

    // Dedup + remove inbox email (including +conv-uuid variants) + remove contact email from CC
    const clean = (list, excludeExtra) => {
        const excludeLower = (excludeExtra || []).map(e => e.toLowerCase())
        return Array.from(new Set(list.filter(email => {
            if (!email) return false
            const lower = email.toLowerCase()
            if (stripConvUUID(lower) === inboxEmail?.toLowerCase()) return false
            if (excludeLower.includes(lower)) return false
            return true
        })))
    }

    return {
        to: clean(toList),
        cc: clean(ccList, toList),
        // BCC stays empty — user is supposed to add it manually.
        bcc: [],
    }
}
