import { toTypedSchema } from '@vee-validate/zod'
import * as z from 'zod'

export function createMessengerFormSchema(t) {
  return z.object({
    name: z.string().min(1, t('globals.messages.fieldRequired', { name: 'Name' })),
    page_id: z.string().optional(),
    ig_account_id: z.string().optional(),
    page_access_token: z.string().min(1, t('globals.messages.fieldRequired', { name: 'Page Access Token' })),
    app_secret: z.string().optional(),
    verify_token: z.string().min(1, t('globals.messages.fieldRequired', { name: 'Verify Token' })),
    auto_assign_on_reply: z.boolean().optional(),
    enabled: z.boolean().optional()
  }).refine(
    (data) => data.page_id || data.ig_account_id,
    { message: 'Either Page ID or Instagram Account ID is required', path: ['page_id'] }
  )
}

export function getMessengerTypedSchema(t) {
  return toTypedSchema(createMessengerFormSchema(t))
}
