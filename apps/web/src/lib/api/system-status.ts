import "server-only";

const defaultAPIBaseURL = "http://127.0.0.1:8080";
const requestTimeoutMilliseconds = 2_500;
const maximumResponseBodyBytes = 64 * 1024;

export type SystemStatusKind =
  | "ready"
  | "database-unavailable"
  | "api-unavailable"
  | "unexpected-response";

export type SystemStatus = {
  readonly kind: SystemStatusKind;
  readonly label: string;
  readonly description: string;
};

type FetchImplementation = (
  input: string | URL | Request,
  init?: RequestInit,
) => Promise<Response>;

export async function getSystemStatus(
  fetchImplementation: FetchImplementation = fetch,
  configuredBaseURL = process.env.SYSAP_API_BASE_URL ?? defaultAPIBaseURL,
): Promise<SystemStatus> {
  const baseURL = parseBaseURL(configuredBaseURL);
  if (baseURL === null) {
    return unexpectedResponse();
  }

  try {
    const healthResponse = await requestEndpoint(
      baseURL,
      "/healthz",
      fetchImplementation,
    );
    if (!isHealthResponse(healthResponse)) {
      return healthResponse.responseOK
        ? unexpectedResponse()
        : apiUnavailable();
    }

    const readinessResponse = await requestEndpoint(
      baseURL,
      "/readyz",
      fetchImplementation,
    );

    if (isReadyResponse(readinessResponse)) {
      return {
        kind: "ready",
        label: "API online · banco pronto",
        description: "Serviços de fundação disponíveis.",
      };
    }

    if (isNotReadyResponse(readinessResponse)) {
      return {
        kind: "database-unavailable",
        label: "API online · banco indisponível",
        description: "O painel demonstrativo continua disponível.",
      };
    }

    return unexpectedResponse();
  } catch {
    return apiUnavailable();
  }
}

type EndpointResult = {
  readonly status: number;
  readonly responseOK: boolean;
  readonly body: unknown;
};

async function requestEndpoint(
  baseURL: URL,
  path: string,
  fetchImplementation: FetchImplementation,
): Promise<EndpointResult> {
  const response = await fetchImplementation(new URL(path, baseURL), {
    cache: "no-store",
    headers: { Accept: "application/json" },
    redirect: "error",
    signal: AbortSignal.timeout(requestTimeoutMilliseconds),
  });

  if (response.redirected) {
    await cancelResponseBody(response.body);
    throw new Error("redirected responses are not accepted");
  }

  const contentType = response.headers.get("content-type") ?? "";
  if (!contentType.toLowerCase().includes("application/json")) {
    await cancelResponseBody(response.body);
    return { status: response.status, responseOK: response.ok, body: null };
  }

  const body = await parseLimitedJSON(response);
  return { status: response.status, responseOK: response.ok, body };
}

async function parseLimitedJSON(response: Response): Promise<unknown> {
  const contentLength = response.headers.get("content-length");
  if (contentLengthExceedsLimit(contentLength)) {
    await cancelResponseBody(response.body);
    throw new Error("response body exceeds the allowed size");
  }

  if (response.body === null) {
    throw new Error("response body is empty");
  }

  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let receivedBytes = 0;

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }

      receivedBytes += value.byteLength;
      if (receivedBytes > maximumResponseBodyBytes) {
        await cancelReader(reader);
        throw new Error("response body exceeds the allowed size");
      }

      chunks.push(value);
    }
  } finally {
    reader.releaseLock();
  }

  const bodyBytes = new Uint8Array(receivedBytes);
  let offset = 0;
  for (const chunk of chunks) {
    bodyBytes.set(chunk, offset);
    offset += chunk.byteLength;
  }

  return JSON.parse(new TextDecoder().decode(bodyBytes)) as unknown;
}

function contentLengthExceedsLimit(contentLength: string | null): boolean {
  if (contentLength === null || !/^\d+$/.test(contentLength)) {
    return false;
  }

  return Number(contentLength) > maximumResponseBodyBytes;
}

async function cancelResponseBody(body: ReadableStream<Uint8Array> | null): Promise<void> {
  if (body === null) {
    return;
  }

  try {
    await body.cancel();
  } catch {
    // The safe unavailable state is more important than a cancellation error.
  }
}

async function cancelReader(reader: ReadableStreamDefaultReader<Uint8Array>): Promise<void> {
  try {
    await reader.cancel();
  } catch {
    // The safe unavailable state is more important than a cancellation error.
  }
}

function parseBaseURL(value: string): URL | null {
  try {
    const url = new URL(value);
    if ((url.protocol !== "http:" && url.protocol !== "https:") || url.username || url.password) {
      return null;
    }
    return url;
  } catch {
    return null;
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function isHealthResponse(result: EndpointResult): boolean {
  return (
    result.status === 200 &&
    result.responseOK &&
    isRecord(result.body) &&
    result.body.status === "ok" &&
    result.body.service === "sysap-api"
  );
}

function isReadyResponse(result: EndpointResult): boolean {
  return (
    result.status === 200 &&
    result.responseOK &&
    isRecord(result.body) &&
    result.body.status === "ready" &&
    result.body.service === "sysap-api" &&
    isRecord(result.body.checks) &&
    result.body.checks.database === "up"
  );
}

function isNotReadyResponse(result: EndpointResult): boolean {
  return (
    result.status === 503 &&
    !result.responseOK &&
    isRecord(result.body) &&
    isRecord(result.body.error) &&
    result.body.error.code === "service_not_ready" &&
    result.body.error.message === "service is not ready"
  );
}

function apiUnavailable(): SystemStatus {
  return {
    kind: "api-unavailable",
    label: "API indisponível",
    description: "Os dados demonstrativos permanecem acessíveis.",
  };
}

function unexpectedResponse(): SystemStatus {
  return {
    kind: "unexpected-response",
    label: "Resposta inesperada da API",
    description: "Os dados demonstrativos permanecem acessíveis.",
  };
}
