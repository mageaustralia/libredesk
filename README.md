<a href="https://zerodha.tech"><img src="https://zerodha.tech/static/images/github-badge.svg" align="right" alt="Zerodha Tech Badge" /></a>


# Libredesk

Modern, open source, self-hosted customer support desk. Single binary app. 

![image](https://libredesk.io/hero.png)


Visit [libredesk.io](https://libredesk.io) for more info. Check out the [**Live demo**](https://demo.libredesk.io/).

## Features

- **Multi Shared Inbox**  
  Libredesk supports multiple shared inboxes, letting you manage conversations across teams effortlessly.
- **Granular Permissions**  
  Create custom roles with granular permissions for teams and individual agents.
- **Smart Automation**  
  Eliminate repetitive tasks with powerful automation rules. Auto-tag, assign, and route conversations based on custom conditions.
- **CSAT Surveys**  
  Measure customer satisfaction with automated surveys.
- **Macros**  
  Save frequently sent messages as templates. With one click, send saved responses, set tags, and more.
- **Smart Organization**  
  Keep conversations organized with tags, custom statuses for conversations, and snoozing. Find any conversation instantly from the search bar.
- **Auto Assignment**  
  Distribute workload with auto assignment rules. Auto-assign conversations based on agent capacity or custom criteria.
- **SLA Management**  
  Set and track response time targets. Get notified when conversations are at risk of breaching SLA commitments.
- **Custom attributes**  
  Create custom attributes for contacts or conversations such as the subscription plan or the date of their first purchase. 
- **AI-Assist**
  Instantly rewrite responses with AI to make them more friendly, professional, or polished.
- **AI-Powered Responses (RAG)**
  Generate context-aware responses using your knowledge base. Indexes FAQ pages and macros for intelligent retrieval.
- **Activity logs**  
  Track all actions performed by agents and admins—updates and key events across the system—for auditing and accountability.
- **Webhooks**  
  Integrate with external systems using real-time HTTP notifications for conversation and message events.
- **Command Bar**  
  Opens with a simple shortcut (CTRL+K) and lets you quickly perform actions on conversations.

And more checkout - [libredesk.io](https://libredesk.io)

---

## Fork Enhancements

This fork ([Trabulium/libredesk](https://github.com/Trabulium/libredesk)) extends upstream Libredesk with the following additions:

### OpenRouter AI Provider

Support for [OpenRouter](https://openrouter.ai/) as an AI provider, giving access to 100+ models (GPT-4o, Claude, Llama, Mistral, etc.) through a single API key.

- **Settings**: Admin → AI Settings → Add OpenRouter provider
- **Files**: `internal/ai/openrouter.go`, `internal/ai/provider.go`, `internal/ai/ai.go`, `frontend/src/views/admin/ai/AISettings.vue`

### Ecommerce Integration (Maho Commerce)

Pull customer and order data from a Maho Commerce (Magento-compatible) store into AI-generated responses. When an agent clicks "Generate Response", the system automatically:

- Looks up the customer by email address
- Fetches their recent orders with item details, prices, and quantities
- Scans conversation messages for order numbers and fetches those specifically
- Includes order status history and shipment tracking with carrier-specific tracking URLs
- Supported carriers: Australia Post, Couriers Please, Team Global Express

**Settings**: Admin → Ecommerce Settings (store URL, OAuth2 credentials)

**Files**:
- `internal/ecommerce/` — Manager, models, and provider interface
- `internal/ecommerce/magento1/` — Maho Commerce API client (OAuth2, Hydra/JSON-LD collections)
- `cmd/ecommerce.go` — API handlers
- `frontend/src/views/admin/ecommerce/EcommerceSettings.vue` — Settings UI

### RAG AI Assistant Enhancements

Improvements to the built-in RAG AI assistant:

- **Knowledge Sources UI**: Admin page to manage knowledge sources (webpages, macros) at Admin → Knowledge Sources
- **Context limiting**: Conversations are trimmed to the last 10 messages (frontend) and 6000 characters (backend) to prevent timeouts on long email threads
- **Ecommerce context injection**: Order and customer data from the ecommerce integration is included in the AI prompt alongside knowledge base results
- **Extended timeouts**: AI provider HTTP timeouts increased to 60 seconds to handle large prompts; frontend request timeout set to 60 seconds

**Files**: `cmd/rag.go`, `internal/rag/`, `frontend/src/features/conversation/ReplyBox.vue`

### UI Customisations

- **Ticket ID in header**: Conversation header shows contact name, ticket reference number, and subject (e.g., "Matthew Campbell #105 - Order enquiry")
- **Simplified sidebar name**: Sidebar shows only the contact name without ticket details to avoid text overflow
- **Self-assign notification suppression**: Assigning a conversation to yourself no longer triggers a notification

**Files**: `frontend/src/stores/conversation.js`, `frontend/src/features/conversation/sidebar/ConversationSideBarContact.vue`, `internal/conversation/conversation.go`

### Per-Inbox Signatures

Each inbox can have its own email signature configured in the inbox settings, appended to outgoing emails.

**Files**: `frontend/src/views/admin/inbox/EditInbox.vue`

---

## Installation

### Docker

The latest image is available on DockerHub at [`libredesk/libredesk:latest`](https://hub.docker.com/r/libredesk/libredesk/tags?page=1&ordering=last_updated&name=latest)

```shell
# Download the compose file and sample config file in the current directory.
curl -LO https://github.com/abhinavxd/libredesk/raw/main/docker-compose.yml
curl -LO https://github.com/abhinavxd/libredesk/raw/main/config.sample.toml

# Copy the config.sample.toml to config.toml and edit it as needed.
cp config.sample.toml config.toml

# Run the services in the background.
docker compose up -d

# Setting System user password.
docker exec -it libredesk_app ./libredesk --set-system-user-password
```

Go to `http://localhost:9000` and login with username `System` and the password you set using the `--set-system-user-password` command.

See [installation docs](https://docs.libredesk.io/getting-started/installation)

__________________

### Binary
- Download the [latest release](https://github.com/abhinavxd/libredesk/releases) and extract the libredesk binary.
- Copy config.sample.toml to config.toml and edit as needed.
- `./libredesk --install` to setup the Postgres DB (or `--upgrade` to upgrade an existing DB. Upgrades are idempotent and running them multiple times have no side effects).
- Run `./libredesk --set-system-user-password` to set the password for the System user.
- Run `./libredesk` and visit `http://localhost:9000` and login with username `System` and the password you set using the --set-system-user-password command.

See [installation docs](https://docs.libredesk.io/getting-started/installation)
__________________

### AI-Powered Responses (RAG)

The AI assistant uses PostgreSQL with pgvector for semantic search.

**Docker:** Already included - uses `pgvector/pgvector:pg17` image.

**Binary/Manual Install:** Install the pgvector extension:
- Ubuntu/Debian: `apt install postgresql-17-pgvector`
- Or compile from [pgvector/pgvector](https://github.com/pgvector/pgvector)

The extension is automatically enabled during database migration.

__________________


## Developers
If you are interested in contributing, refer to the [developer setup](https://docs.libredesk.io/contributing/developer-setup). The backend is written in Go and the frontend is Vue js 3 with Shadcn for UI components.


## Translators
You can help translate Libredesk into your language on [Crowdin](https://crowdin.com/project/libredesk).
