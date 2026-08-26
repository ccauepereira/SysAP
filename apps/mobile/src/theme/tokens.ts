/**
 * SysAP Design Tokens (Artur Performance brand)
 * Source of truth: assets/brand/README.md
 */

export const brandTokens = {
  colors: {
    // Brand Gold
    brandGold300: "#EAC855",
    brandGold500: "#D4AE29",
    brandGold700: "#B08A19",

    // Neutrals & Surfaces
    surface950: "#080A0C", // Main background
    surface900: "#0D1117", // Cards & elevated containers
    surface800: "#161B22", // Input backgrounds, item cards
    surface700: "#21262D", // Active/hovered elements

    // Borders
    borderDefault: "#30363D",
    borderFocus: "#D4AE29",
    borderError: "#F85149",

    // Text
    textPrimary: "#F7F7F5",
    textSecondary: "#8B949E",
    textMuted: "#6E7681",
    textOnGold: "#111318", // Dark text on gold buttons

    // Status
    statusError: "#F85149",
    statusErrorBg: "#2E1517",
    statusErrorBorder: "#8E2B2B",
    statusSuccess: "#3FB950",
    statusWarning: "#D29922",
  },
  spacing: {
    xs: 4,
    sm: 8,
    md: 16,
    lg: 24,
    xl: 32,
    xxl: 48,
  },
  borderRadius: {
    sm: 4,
    md: 8,
    lg: 12,
    xl: 16,
    full: 9999,
  },
  typography: {
    title: {
      fontSize: 22,
      fontWeight: "700" as const,
      lineHeight: 28,
    },
    subtitle: {
      fontSize: 15,
      fontWeight: "600" as const,
      lineHeight: 22,
    },
    body: {
      fontSize: 14,
      fontWeight: "400" as const,
      lineHeight: 20,
    },
    button: {
      fontSize: 15,
      fontWeight: "600" as const,
      lineHeight: 20,
    },
    caption: {
      fontSize: 12,
      fontWeight: "400" as const,
      lineHeight: 16,
    },
  },
} as const;
