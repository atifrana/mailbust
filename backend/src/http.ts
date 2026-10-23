/** Small JSON response helpers shared by the Worker and Durable Object. */

const BASE_HEADERS = { "content-type": "application/json; charset=utf-8" };

export function json(data: unknown, status = 200): Response {
  return new Response(JSON.stringify(data), { status, headers: BASE_HEADERS });
}

export function error(status: number, code: string, message: string): Response {
  return json({ error: { code, message } }, status);
}
