# Home Page Redesign Plan

## Overview
Redesign the boom-prototype frontend home page with professional, minimalist styling using the provided color palette. Remove "Boom" branding and add appointment request functionality.

## Color Palette
- **Primary Blue:** #2B88D9 (Ticino Blue)
- **Secondary Blue:** #6394BF (Kentucky)
- **Teal Accent:** #0396A6 (Brilliant)
- **Vermillion:** #D93D1A (CTA/accent)
- **Red:** #F23D3D (Vivaldi Red)
- **NO PURPLE**

## File to Modify
- `frontend/src/pages/Home.tsx` - Complete redesign

## Design Changes

### 1. Remove Boom Branding
- Replace "Boom" title with generic "Meet" or icon-based logo
- Remove "Built with LiveKit" footer reference

### 2. New Layout Structure
```
┌─────────────────────────────────────────┐
│ Header: Logo + Sign In button           │
├─────────────────────────────────────────┤
│ Hero: Welcome headline + subtitle       │
├─────────────────────────────────────────┤
│ Main Card (tabbed):                     │
│ ┌─────────────┬─────────────────────┐   │
│ │ Join Meeting│ Request Appointment │   │
│ └─────────────┴─────────────────────┘   │
│                                         │
│ Tab 1: Join Meeting                     │
│ - Name input                            │
│ - Meeting ID input + Join button        │
│ - "or start new" divider                │
│ - Start New Meeting button              │
│                                         │
│ Tab 2: Appointment Request Form         │
│ - Name, Email (2-col)                   │
│ - Date, Time (2-col)                    │
│ - Message textarea                      │
│ - Submit button                         │
├─────────────────────────────────────────┤
│ Features: 3 icons (Secure, Transcribed, │
│           Group Ready)                  │
├─────────────────────────────────────────┤
│ Footer: Minimal tagline                 │
└─────────────────────────────────────────┘
```

### 3. Styling
- Light theme (slate-50/100 background vs current dark)
- White card with shadow
- Tab navigation for Join vs Request
- Color usage:
  - #2B88D9: Primary buttons, active tabs, focus rings
  - #0396A6: Join button, success states
  - #D93D1A: Request Appointment submit button
  - #6394BF: Secondary accents, icons

### 4. New Features
- **Sign In button** (placeholder in header)
- **Tabbed interface** switching between:
  - Join Meeting (existing functionality)
  - Request Appointment (new form)
- **Appointment form fields:**
  - Name (required)
  - Email (required)
  - Preferred Date (required)
  - Preferred Time (required)
  - Message (optional)
- **Feature icons section** highlighting capabilities

## Appointment Form Integration
- POST form data to n8n webhook on user's droplet
- n8n URL: https://n8n.obsidianvoice.ai (already running)
- Create new n8n workflow with webhook trigger
- Workflow can notify via Discord/email (user configures in n8n)

## Implementation Steps
1. Update color scheme from dark to light theme
2. Add header with logo icon and Sign In button
3. Add hero section with professional welcome copy
4. Create tabbed card interface
5. Preserve existing join meeting logic in Tab 1
6. Add appointment request form in Tab 2
7. POST form to n8n webhook URL (env var: VITE_APPOINTMENT_WEBHOOK)
8. Add feature icons section
9. Update footer to be minimal
10. Add env var to Vercel for webhook URL

## Notes
- Login button is a placeholder (no auth implemented yet)
- All existing meeting functionality preserved
- User will create n8n workflow to handle notifications
