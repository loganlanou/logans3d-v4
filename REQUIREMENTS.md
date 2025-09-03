

# Logan’s 3D Creations — Project Requirements (v0.1)

_Last updated: Sept 3, 2025_

## 1) Overview
This document captures the scope and requirements for rebuilding the Logan’s 3D Creations website. It is intended to be a living spec that evolves as features are refined and prioritized.

## 2) Goals
- Unified, modern site with integrated catalog and custom order flow.
- Stripe-based checkout (no rush; Square currently handles sales).
- Excellent performance, SEO, and accessibility.
- Easy editing from mobile (CMS/admin), with room to grow post‑launch.

## 3) Non‑Goals (V1)
- Complex B2B quoting automation (only a simple rough estimator in the future).
- Multi‑currency, marketplace, or multi‑vendor flows.

## 4) Tech Stack (updated)
**Core Architecture**: Server-Side Rendered (SSR) with Progressive Enhancement

### Backend & Framework
- **Backend**: Go 1.25
- **Web Framework**: [Echo v4.13+](https://github.com/labstack/echo) - High performance HTTP router
- **Templating**: [a-h/templ](https://github.com/a-h/templ) - Type-safe Go HTML templates
- **Authentication**: JWT + OAuth2 (Google/social login for customer accounts)

### Frontend & Styling  
- **Markup**: Server-rendered HTML (no HTMX - following SSR + Alpine.js pattern)
- **JavaScript**: [Alpine.js 3.x](https://alpinejs.dev/) - Lightweight reactive components
- **Styling**: [Tailwind CSS v4+](https://tailwindcss.com/) with PostCSS processing
- **Progressive Enhancement**: JavaScript enhances server-rendered content

### Database & Code Generation
- **Database**: SQLite with [SQLC](https://sqlc.dev/) for type-safe queries
- **Migrations**: [Goose](https://pressly.github.io/goose/) for database schema management  
- **Database Backup/Replication**: [Litestream](https://litestream.io) for production backup
- **Code Generation**: `go generate` pipeline (SQLC + Templ compilation)

### Development & Build Tools
- **Build System**: Make + [Air](https://github.com/cosmtrek/air) for hot reloading
- **CSS Processing**: PostCSS + Tailwind CLI
- **Testing**: [Playwright](https://playwright.dev/) for E2E + Go testing framework
- **Linting**: [golangci-lint](https://golangci-lint.run/) for Go code quality

### Production & Deployment
- **Hosting**: [Vercel](https://vercel.com) free plan with automatic Git deployments
- **Runtime**: Vercel's Go serverless functions
- **Domain**: logans3dcreations.com (transfer from Square to DNSimple required)
- **SSL**: Automatic HTTPS with Vercel's Edge Network  
- **Payments**: Stripe Checkout (embedded), Apple Pay/Google Pay enabled
- **Email**: Transactional email provider (SendGrid, Mailgun, etc.)
- **Monitoring**: Vercel Analytics + structured logging

## 5) Information Architecture (top-level)
- Home, Shop, Custom Printing, Portfolio, Events, About, FAQ, Contact, Cart/Checkout
- Legal: Privacy, Terms, Shipping & Returns, Custom Work Policy

## 6) Feature Master Table
Below is the canonical feature list. Columns: **Feature**, **Category**, **Priority**, **Phase**, **Notes**.

| Feature                                              | Category             | Priority     | Phase    | Notes                                                            |
|:-----------------------------------------------------|:---------------------|:-------------|:---------|:-----------------------------------------------------------------|
| Product catalog (grid)                               | Storefront & Catalog | Must-have    | Phase 2  | Browse products in a responsive grid                             |
| Product categories                                   | Storefront & Catalog | Must-have    | Phase 2  | Taxonomy: Multicolor, Single Color, Dinosaurs, Desk, Parts, etc. |
| Product sorting: price (low→high, high→low)          | Storefront & Catalog | Must-have    | Phase 2  | Sort by price both directions                                    |
| Product sorting: size                                | Storefront & Catalog | Must-have    | Phase 2  | Sort by size option                                              |
| Product sorting: lead time                           | Storefront & Catalog | Must-have    | Phase 2  | Sort by estimated fulfillment time                               |
| Product filter: category                             | Storefront & Catalog | Must-have    | Phase 2  | Filter by product category                                       |
| Product filter: material/color                       | Storefront & Catalog | Must-have    | Phase 2  | Filter by Multicolor vs Single Color, material                   |
| Search bar (basic)                                   | Storefront & Catalog | Must-have    | Phase 2  | Keyword search across products                                   |
| Product detail page (PDP)                            | Storefront & Catalog | Must-have    | Phase 2  | Images, description, price                                       |
| Variants (size/material/color)                       | Storefront & Catalog | Must-have    | Phase 2  | Selectable options on PDP                                        |
| Lead time display                                    | Storefront & Catalog | Must-have    | Phase 2  | Show estimated production time                                   |
| Related products                                     | Storefront & Catalog | Nice-to-have | Phase 4+ | Cross-sell/upsell on PDP                                         |
| Wishlist / Favorites                                 | Storefront & Catalog | Nice-to-have | Phase 4+ | Save items for later                                             |
| Autocomplete search / K-menu                         | Storefront & Catalog | Nice-to-have | Phase 4+ | Spotlight-style global search                                    |
| Add to cart                                          | Cart & Checkout      | Must-have    | Phase 2  | Cart drawer + cart page                                          |
| Cart drawer (mini-cart)                              | Cart & Checkout      | Must-have    | Phase 2  | Slide-out cart with line items                                   |
| Shipping estimator                                   | Cart & Checkout      | Must-have    | Phase 3  | Estimate based on address/ZIP                                    |
| Local tax estimate                                   | Cart & Checkout      | Must-have    | Phase 3  | Tax calculated pre-checkout                                      |
| Local pickup option                                  | Cart & Checkout      | Must-have    | Phase 3  | Pickup at events/store                                           |
| Stripe checkout integration                          | Cart & Checkout      | Must-have    | Phase 3  | Cards + Apple/Google Pay                                         |
| Discount/promo codes                                 | Cart & Checkout      | Nice-to-have | Phase 4+ | Apply code at cart/checkout                                      |
| Order confirmation page                              | Cart & Checkout      | Must-have    | Phase 3  | Shows order details post-payment                                 |
| Order confirmation email                             | Cart & Checkout      | Must-have    | Phase 3  | Receipt sent to customer                                         |
| Customer accounts (Google login)                     | Cart & Checkout      | Nice-to-have | Phase 4+ | Sign in with Google, view orders                                 |
| Order history & tracking                             | Cart & Checkout      | Nice-to-have | Phase 4+ | Customers can review past orders                                 |
| Subscriptions / recurring                            | Cart & Checkout      | Nice-to-have | Phase 4+ | For consumables or club items                                    |
| Loyalty / rewards                                    | Cart & Checkout      | Nice-to-have | Phase 4+ | Points or discounts for repeat buyers                            |
| Custom order page                                    | Custom Printing      | Must-have    | Phase 3  | Dedicated route with details & form                              |
| Quote form: contact fields                           | Custom Printing      | Must-have    | Phase 3  | Name, email, phone                                               |
| Quote form: project details                          | Custom Printing      | Must-have    | Phase 3  | Description, dimensions, quantity, deadline, budget              |
| Quote form: material/finish selectors                | Custom Printing      | Must-have    | Phase 3  | PLA, PETG, ABS/ASA, Resin; raw/sanded/paint-ready                |
| File uploads (.STL/.OBJ/.STEP/.3MF)                  | Custom Printing      | Must-have    | Phase 3  | Up to 5 files, 100MB each + reference images                     |
| Rough cost estimator (non-binding)                   | Custom Printing      | Nice-to-have | Phase 4+ | Simple calculator based on time/material/finish                  |
| Admin notifications for quotes                       | Custom Printing      | Must-have    | Phase 3  | Email/Slack alert with links to files                            |
| Customer confirmation email (quote)                  | Custom Printing      | Must-have    | Phase 3  | Acknowledges receipt of request                                  |
| Stripe payment link / deposit (post-quote)           | Custom Printing      | Nice-to-have | Phase 4+ | Collect partial/full payment after approval                      |
| Order status tracker                                 | Custom Printing      | Nice-to-have | Phase 4+ | Queued → Printing → Finishing → Ready                            |
| Home page (hero, featured categories)                | Content & Trust      | Must-have    | Phase 1  | Clear CTA to Shop + Custom Quote                                 |
| About page (story & shop tour)                       | Content & Trust      | Must-have    | Phase 1  | Your story, photos                                               |
| About the 3D printing process                        | Content & Trust      | Must-have    | Phase 1  | Educational page                                                 |
| Portfolio / Gallery                                  | Content & Trust      | Nice-to-have | Phase 2  | Case studies with tags & images                                  |
| Testimonials / reviews                               | Content & Trust      | Nice-to-have | Phase 4+ | Product reviews + general testimonials                           |
| Events page (current events list)                    | Content & Trust      | Must-have    | Phase 1  | Upcoming shows with dates & locations                            |
| Events calendar sync (ICS)                           | Content & Trust      | Nice-to-have | Phase 4+ | Add-to-calendar download links                                   |
| Blog / Updates                                       | Content & Trust      | Nice-to-have | Phase 4+ | News, guides, SEO content                                        |
| Contact page (form + map + hours + socials)          | Contact & Engagement | Must-have    | Phase 1  | Email form + embedded map                                        |
| Live social feed embeds                              | Contact & Engagement | Nice-to-have | Phase 4+ | Instagram/Facebook/TikTok feed                                   |
| Newsletter sign-up                                   | Contact & Engagement | Nice-to-have | Phase 4+ | Email marketing integration                                      |
| 3D printer support tickets                           | Support & Community  | Nice-to-have | Phase 4+ | Issue submission & tracking                                      |
| Mobile-first responsive design                       | Core Infrastructure  | Must-have    | Phase 1  | Works great on phones and desktops                               |
| Admin/CMS (phone editing)                            | Core Infrastructure  | Must-have    | Phase 1  | Add/edit products & content from phone                           |
| SEO basics (meta, sitemap, schema)                   | Core Infrastructure  | Must-have    | Phase 1  | Product/Org/LocalBusiness schema                                 |
| Analytics (GA4 + ecommerce events)                   | Core Infrastructure  | Must-have    | Phase 1  | Track views/cart/checkout/purchase                               |
| Performance budgets                                  | Core Infrastructure  | Must-have    | Phase 1  | Lighthouse ≥90; image optimization                               |
| Accessibility (WCAG)                                 | Core Infrastructure  | Must-have    | Phase 1  | Alt text, focus rings, keyboard nav, contrast                    |
| Legal pages (Privacy, Terms, Returns, Custom Policy) | Core Infrastructure  | Must-have    | Phase 1  | Compliant policy pages                                           |
| Domain transfer from Square to DNSimple              | Core Infrastructure  | Must-have    | Pre-Phase 1 | Transfer logans3dcreations.com domain registration             |
| DNS configuration for Vercel deployment              | Core Infrastructure  | Must-have    | Pre-Phase 1 | Setup DNS records for Vercel hosting                           |

## 7) Launch Phases (high level)
**Pre-Phase 1 — Domain & Infrastructure Setup**: Transfer logans3dcreations.com from Square to DNSimple, configure DNS for Vercel deployment, setup development and staging environments.  
**Phase 1 — Foundation & Content**: Global layout, Home, About, Events, Contact, Legal, CMS setup.  
**Phase 2 — Storefront & Catalog**: PLP grid, filters/sorting, PDP, cart (drawer + page).  
**Phase 3 — Checkout & Custom Orders**: Stripe checkout, shipping/tax estimate, custom order form + uploads/notifications.  
**Phase 4+ — Enhancements**: Wishlist, Google login, reviews/testimonials, portfolio deep dives, live social feed, order status tracker, advanced search.

## 8) Performance & Accessibility
- Performance budgets: LCP < 3s on 4G; JS ≤ 200KB gz on landing; images AVIF/WebP + lazy‑load.
- Accessibility: keyboard navigable menus, visible focus, ARIA labels, contrast ≥ 4.5:1, alt text.

## 9) Analytics & SEO
- GA4 + ecommerce events (view_item_list → purchase) and quote funnel events.
- SEO: clean URLs, metadata, sitemap.xml, robots.txt, Product/Organization/LocalBusiness schema.

## 10) Content Inputs (from Logan)
- Brand assets (logo SVG, colors), 12–24 product photos, 6–12 portfolio items.
- Policy copy (privacy, terms, returns, custom policy), testimonials, event list (3–6 months).

---

**Change Log**
- v0.1 (2025‑09‑03): Initial requirements + embedded features table.