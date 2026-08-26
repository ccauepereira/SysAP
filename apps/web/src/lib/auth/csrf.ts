import "server-only";

export function validateOrigin(request: Request): boolean {
  const origin = request.headers.get("origin") || request.headers.get("referer");
  const allowedOrigin = process.env.SYSAP_WEB_ORIGIN;

  if (!allowedOrigin) {
    // If not configured, deny all mutating requests for safety
    return false;
  }

  if (!origin) {
    return false;
  }

  try {
    const originUrl = new URL(origin);
    const allowedUrl = new URL(allowedOrigin);
    return originUrl.origin === allowedUrl.origin;
  } catch {
    return false;
  }
}
