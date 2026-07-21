import { createServer } from "node:http";
import { describe, expect, it, vi } from "vitest";
import { getSystemStatus, type SystemStatus } from "./system-status";

const baseURL = "http://api.example.invalid";
const responseLimitBytes = 64 * 1024;
const validHealth = { status: "ok", service: "sysap-api" };
const validReadiness = {
  status: "ready",
  service: "sysap-api",
  checks: { database: "up" },
};

function jsonResponse(
  body: unknown,
  status = 200,
  headers: HeadersInit = {},
): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      "content-type": "application/json; charset=utf-8",
      ...headers,
    },
  });
}

function readyFetcher() {
  return vi
    .fn()
    .mockResolvedValueOnce(jsonResponse(validHealth))
    .mockResolvedValueOnce(jsonResponse(validReadiness));
}

function exactSizeJSON(body: Record<string, unknown>, size: number): string {
  const emptyPadding = JSON.stringify({ ...body, padding: "" });
  const paddingBytes = size - new TextEncoder().encode(emptyPadding).byteLength;
  if (paddingBytes < 0) {
    throw new Error("target size is smaller than the JSON contract");
  }

  const result = JSON.stringify({ ...body, padding: "x".repeat(paddingBytes) });
  if (new TextEncoder().encode(result).byteLength !== size) {
    throw new Error("could not create the requested JSON fixture size");
  }
  return result;
}

function streamingOversizedResponse(contentLength?: string) {
  let chunksProduced = 0;
  let canceled = false;
  const chunkSize = 24 * 1024;
  const stream = new ReadableStream<Uint8Array>(
    {
      pull(controller) {
        chunksProduced += 1;
        controller.enqueue(new Uint8Array(chunkSize));
      },
      cancel() {
        canceled = true;
      },
    },
    { highWaterMark: 0 },
  );
  const headers: HeadersInit = { "content-type": "application/json" };
  if (contentLength !== undefined) {
    headers["content-length"] = contentLength;
  }

  return {
    response: new Response(stream, { headers }),
    telemetry: {
      get canceled() {
        return canceled;
      },
      get chunksProduced() {
        return chunksProduced;
      },
    },
  };
}

function expectSafeResult(
  result: SystemStatus,
  forbiddenValues: readonly string[],
): void {
  const serialized = JSON.stringify(result);
  for (const value of forbiddenValues) {
    expect(serialized).not.toContain(value);
  }
}

describe("getSystemStatus", () => {
  it("accepts a valid health response before checking readiness", async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(validHealth))
      .mockResolvedValueOnce(
        jsonResponse(
          {
            error: {
              code: "service_not_ready",
              message: "service is not ready",
              request_id: "internal-request-id",
            },
          },
          503,
        ),
      );

    await expect(getSystemStatus(fetcher, baseURL)).resolves.toMatchObject({
      kind: "database-unavailable",
    });
    expect(fetcher).toHaveBeenCalledTimes(2);
    expect(fetcher.mock.calls[0]?.[0]).toEqual(new URL("/healthz", baseURL));
  });

  it("accepts a valid readiness response", async () => {
    const fetcher = readyFetcher();

    await expect(getSystemStatus(fetcher, baseURL)).resolves.toMatchObject({
      kind: "ready",
      label: "API online · banco pronto",
    });
    expect(fetcher.mock.calls[1]?.[0]).toEqual(new URL("/readyz", baseURL));
    expect(fetcher.mock.calls[0]?.[1]).toMatchObject({
      cache: "no-store",
      redirect: "error",
    });
  });

  it("reports the expected ready 503 contract as database unavailable", async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse(validHealth))
      .mockResolvedValueOnce(
        jsonResponse(
          {
            error: {
              code: "service_not_ready",
              message: "service is not ready",
              request_id: "internal-request-id",
            },
          },
          503,
        ),
      );

    const result = await getSystemStatus(fetcher, baseURL);

    expect(result.kind).toBe("database-unavailable");
    expectSafeResult(result, ["internal-request-id", baseURL]);
  });

  it("aborts a request that exceeds the timeout", async () => {
    const fetcher = vi.fn(
      (_input: string | URL | Request, init?: RequestInit) =>
        new Promise<Response>((_resolve, reject) => {
          init?.signal?.addEventListener(
            "abort",
            () => reject(new Error("private timeout detail")),
            { once: true },
          );
        }),
    );

    const result = await getSystemStatus(fetcher, baseURL);

    expect(result.kind).toBe("api-unavailable");
    expect(fetcher.mock.calls[0]?.[1]?.signal?.aborted).toBe(true);
    expectSafeResult(result, ["private timeout detail", baseURL]);
  }, 5_000);

  it("treats HTTP 500 as API unavailable", async () => {
    const result = await getSystemStatus(
      vi.fn().mockResolvedValue(jsonResponse({ error: "private" }, 500)),
      baseURL,
    );

    expect(result.kind).toBe("api-unavailable");
    expectSafeResult(result, ["private", baseURL]);
  });

  it("rejects invalid JSON safely", async () => {
    const response = new Response("{invalid private body", {
      headers: { "content-type": "application/json" },
    });

    const result = await getSystemStatus(
      vi.fn().mockResolvedValue(response),
      baseURL,
    );

    expect(result.kind).toBe("api-unavailable");
    expectSafeResult(result, ["invalid private body", baseURL]);
  });

  it("rejects an empty JSON body safely", async () => {
    const response = new Response("", {
      headers: { "content-type": "application/json" },
    });

    await expect(
      getSystemStatus(vi.fn().mockResolvedValue(response), baseURL),
    ).resolves.toMatchObject({ kind: "api-unavailable" });
  });

  it("treats an incorrect content type as unexpected", async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValue(new Response("private upstream body", { status: 200 }));

    const result = await getSystemStatus(fetcher, baseURL);

    expect(result.kind).toBe("unexpected-response");
    expectSafeResult(result, ["private upstream body", baseURL]);
  });

  it("rejects Content-Length above 64 KiB before reading", async () => {
    const cancel = vi.fn().mockResolvedValue(undefined);
    const getReader = vi.fn(() => {
      throw new Error("body must not be read");
    });
    const response = {
      body: { cancel, getReader },
      headers: new Headers({
        "content-length": String(responseLimitBytes + 1),
        "content-type": "application/json",
      }),
      ok: true,
      redirected: false,
      status: 200,
    } as unknown as Response;

    const result = await getSystemStatus(
      vi.fn().mockResolvedValue(response),
      baseURL,
    );

    expect(result.kind).toBe("api-unavailable");
    expect(cancel).toHaveBeenCalledOnce();
    expect(getReader).not.toHaveBeenCalled();
  });

  it("cancels a body above 64 KiB without Content-Length", async () => {
    const { response, telemetry } = streamingOversizedResponse();

    const result = await getSystemStatus(
      vi.fn().mockResolvedValue(response),
      baseURL,
    );

    expect(result.kind).toBe("api-unavailable");
    expect(telemetry.canceled).toBe(true);
    expect(telemetry.chunksProduced).toBe(3);
  });

  it("does not trust a Content-Length smaller than the streamed body", async () => {
    const { response, telemetry } = streamingOversizedResponse("128");

    const result = await getSystemStatus(
      vi.fn().mockResolvedValue(response),
      baseURL,
    );

    expect(result.kind).toBe("api-unavailable");
    expect(telemetry.canceled).toBe(true);
    expect(telemetry.chunksProduced).toBe(3);
  });

  it("accepts a valid body exactly at the 64 KiB limit", async () => {
    const exactBody = exactSizeJSON(validHealth, responseLimitBytes);
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(exactBody, {
          headers: {
            "content-length": String(responseLimitBytes),
            "content-type": "application/json",
          },
        }),
      )
      .mockResolvedValueOnce(jsonResponse(validReadiness));

    await expect(getSystemStatus(fetcher, baseURL)).resolves.toMatchObject({
      kind: "ready",
    });
  });

  it("accepts small legitimate contracts below the limit", async () => {
    const fetcher = readyFetcher();

    await expect(getSystemStatus(fetcher, baseURL)).resolves.toMatchObject({
      kind: "ready",
    });
  });

  it("rejects a redirect without requesting its valid destination", async () => {
    let redirectedDestinationRequests = 0;
    const privateLocation = "/private-valid-health";
    const server = createServer((request, response) => {
      if (request.url === "/healthz") {
        response.writeHead(302, { location: privateLocation });
        response.end();
        return;
      }

      if (request.url === privateLocation) {
        redirectedDestinationRequests += 1;
        response.writeHead(200, { "content-type": "application/json" });
        response.end(JSON.stringify(validHealth));
        return;
      }

      response.writeHead(404);
      response.end();
    });

    await new Promise<void>((resolve, reject) => {
      server.once("error", reject);
      server.listen(0, "127.0.0.1", resolve);
    });

    try {
      const address = server.address();
      if (address === null || typeof address === "string") {
        throw new Error("temporary server did not expose a TCP address");
      }
      const localBaseURL = `http://127.0.0.1:${address.port}`;

      const result = await getSystemStatus(fetch, localBaseURL);

      expect(result.kind).toBe("api-unavailable");
      expect(redirectedDestinationRequests).toBe(0);
      expectSafeResult(result, [privateLocation, localBaseURL]);
    } finally {
      await new Promise<void>((resolve, reject) => {
        server.close((error) => (error === undefined ? resolve() : reject(error)));
      });
    }
  });

  it("defensively rejects a redirected response from an injected fetch", async () => {
    const response = {
      body: null,
      headers: new Headers({ "content-type": "application/json" }),
      ok: true,
      redirected: true,
      status: 200,
    } as unknown as Response;

    await expect(
      getSystemStatus(vi.fn().mockResolvedValue(response), baseURL),
    ).resolves.toMatchObject({ kind: "api-unavailable" });
  });

  it("returns a safe state when the connection is refused", async () => {
    const internalError = "connect ECONNREFUSED at a private host";
    const fetcher = vi.fn().mockRejectedValue(new Error(internalError));

    const result = await getSystemStatus(fetcher, baseURL);

    expect(result.kind).toBe("api-unavailable");
    expectSafeResult(result, [internalError, baseURL]);
  });

  it("rejects an unknown contract without leaking adversarial content", async () => {
    const privateDetail = "private upstream detail";
    const fetcher = vi.fn().mockResolvedValue(
      jsonResponse({ status: "unknown", detail: privateDetail }),
    );

    const result = await getSystemStatus(fetcher, baseURL);

    expect(result.kind).toBe("unexpected-response");
    expectSafeResult(result, [privateDetail, baseURL]);
  });
});
