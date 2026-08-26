/**
 * Environment configuration and URL validation for SysAP Mobile.
 * Rule: EXPO_PUBLIC_API_BASE_URL is public and MUST NOT contain secrets.
 */

const DEFAULT_DEV_API_URL = "http://10.0.2.2:8080";

function isPrivateOrLocalHost(hostname: string): boolean {
  if (
    hostname === "localhost" ||
    hostname === "127.0.0.1" ||
    hostname === "10.0.2.2" || // Android emulator loopback alias
    hostname === "10.0.3.2" || // Genymotion emulator loopback alias
    hostname === "::1"
  ) {
    return true;
  }

  // Check RFC1918 Private IPv4 addresses for physical device testing on local Wi-Fi:
  // 10.0.0.0/8
  if (/^10\.\d{1,3}\.\d{1,3}\.\d{1,3}$/.test(hostname)) return true;
  // 172.16.0.0/12
  if (/^172\.(1[6-9]|2\d|3[0-1])\.\d{1,3}\.\d{1,3}$/.test(hostname)) return true;
  // 192.168.0.0/16
  if (/^192\.168\.\d{1,3}\.\d{1,3}$/.test(hostname)) return true;

  return false;
}

export function validateApiBaseUrl(rawUrl: string, isProduction: boolean = false): string {
  if (!rawUrl || typeof rawUrl !== "string") {
    throw new Error("API base URL is required");
  }

  let parsed: URL;
  try {
    parsed = new URL(rawUrl);
  } catch {
    throw new Error("API base URL is not a valid URL");
  }

  if (isProduction) {
    if (parsed.protocol !== "https:") {
      throw new Error("Production API base URL must use HTTPS");
    }
  } else {
    if (parsed.protocol !== "https:" && parsed.protocol !== "http:") {
      throw new Error("API base URL must use HTTP or HTTPS");
    }
    if (parsed.protocol === "http:" && !isPrivateOrLocalHost(parsed.hostname)) {
      throw new Error(
        "Insecure HTTP is only permitted for localhost or private LAN development addresses",
      );
    }
  }

  // Return origin without trailing slash
  return `${parsed.protocol}//${parsed.host}`;
}

export function getApiBaseUrl(): string {
  const rawUrl = process.env.EXPO_PUBLIC_API_BASE_URL || DEFAULT_DEV_API_URL;
  const isProd = process.env.NODE_ENV === "production";
  return validateApiBaseUrl(rawUrl, isProd);
}
