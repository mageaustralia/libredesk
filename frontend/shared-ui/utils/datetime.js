import { format, differenceInMinutes, differenceInHours, differenceInDays, differenceInMonths, differenceInYears, isToday, isYesterday, isSameYear } from 'date-fns'

export function getRelativeTime (timestamp, now = new Date()) {
  try {
    const mins = differenceInMinutes(now, timestamp)
    const hours = differenceInHours(now, timestamp)
    const days = differenceInDays(now, timestamp)
    const months = differenceInMonths(now, timestamp)
    const years = differenceInYears(now, timestamp)

    if (mins === 0) return 'now'
    if (mins < 60) return `${mins}m`
    if (hours < 24) return `${hours}h`
    if (days < 31) return `${days}d`
    if (months < 12) return `${months}mo`
    return `${years}y`
  } catch (error) {
    console.error('Error parsing time', error, 'timestamp', timestamp)
    return ''
  }
}

export const formatDuration = (seconds, showSeconds = true) => {
  const totalSeconds = Math.floor(seconds)
  if (totalSeconds < 60) return `${totalSeconds}s`
  if (totalSeconds < 3600) return `${Math.floor(totalSeconds / 60)}m ${totalSeconds % 60}s`
  const hours = Math.floor(totalSeconds / 3600)
  const mins = Math.floor((totalSeconds % 3600) / 60)
  const secs = totalSeconds % 60
  return `${hours}h ${mins}m ${showSeconds ? `${secs}s` : ''}`
}

export const formatMessageTimestamp = (time) => {
  const now = new Date()
  const date = new Date(time)
  const mins = differenceInMinutes(now, date)
  const hours = differenceInHours(now, date)
  const days = differenceInDays(now, date)

  let relative
  if (mins < 1) relative = 'Just now'
  else if (mins < 60) relative = mins === 1 ? '1 minute ago' : `${mins} minutes ago`
  else if (hours < 24) relative = hours === 1 ? '1 hour ago' : `${hours} hours ago`
  else if (days < 30) relative = days === 1 ? '1 day ago' : `${days} days ago`
  else relative = null

  const fullDate = format(date, "EEE, d MMM yyyy 'at' h:mm a")
  if (relative) {
    return `${relative} (${fullDate})`
  }
  return fullDate
}

export const formatFullTimestamp = (time) => {
  return format(time, "EEE, d MMM yyyy 'at' h:mm a")
}

// formatActivityTimestamp is the compact label used by activity-log entries in
// the conversation thread. Plain "h:mm a" was misleading for events older than
// today (a 5-day-old "Assigned to X 4:22 PM" reads as today). This grows the
// label just enough to disambiguate, while staying short for inline use:
//   today      → "4:22 PM"
//   yesterday  → "Yesterday, 4:22 PM"
//   this year  → "21 May, 4:22 PM"
//   older      → "21 May 2025, 4:22 PM"
// Callers typically pair this with a tooltip showing formatFullTimestamp.
export const formatActivityTimestamp = (time) => {
  const date = time instanceof Date ? time : new Date(time)
  if (isToday(date)) return format(date, 'h:mm a')
  if (isYesterday(date)) return `Yesterday, ${format(date, 'h:mm a')}`
  if (isSameYear(date, new Date())) return format(date, 'd MMM, h:mm a')
  return format(date, 'd MMM yyyy, h:mm a')
}
