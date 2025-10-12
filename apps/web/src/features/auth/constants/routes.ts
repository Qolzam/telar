export const AUTH_ROUTES = {
  LOGIN: '/login',
  SIGNUP: '/signup',
  LOGOUT: '/logout',
  FORGOT_PASSWORD: '/forgot-password',
  RESET_PASSWORD: '/reset-password',
  VERIFY_EMAIL: '/verify-email',
  DASHBOARD: '/dashboard',
  PROFILE: '/profile',
  SETTINGS: '/settings',
} as const;

export const API_ROUTES = {
  LOGIN: '/api/auth/login',
  SIGNUP: '/api/auth/signup',
  VERIFY: '/api/auth/verify',
  LOGOUT: '/api/auth/logout',
  SESSION: '/api/auth/session',
  FORGOT_PASSWORD: '/api/auth/forgot-password',
  RESET_PASSWORD: '/api/auth/reset-password',
  CHANGE_PASSWORD: '/api/auth/change-password',
} as const;

export type AuthRoute = typeof AUTH_ROUTES[keyof typeof AUTH_ROUTES];
export type ApiRoute = typeof API_ROUTES[keyof typeof API_ROUTES];

