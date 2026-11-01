import { isUnauthorizedError, uploadIcon } from './api';
import type { IconResp } from './types';

const browserIconFetchTimeoutMS = 10000;
const browserIconRequestTimeoutMS = 3000;
const browserIconHTMLReadLimit = 256 * 1024;
const browserIconManifestReadLimit = 64 * 1024;
const browserIconFileReadLimit = 5 * 1024 * 1024;
const browserIconMaxCandidates = 12;
const browserIconMaxPageCandidates = 6;
const browserIconMaxManifestCandidates = 2;
const browserIconMaxManifests = 2;

type BrowserIconResources = {
  icons: URL[];
  manifests: URL[];
  baseURL: URL;
};

type WebManifest = {
  icons?: unknown;
};

export async function fetchAndUploadBrowserIcon(websiteURL: string): Promise<IconResp> {
  const baseURL = normalizeWebsiteURL(websiteURL);
  const deadline = Date.now() + browserIconFetchTimeoutMS;
  const resources = await loadPageIconResources(baseURL, deadline);
  const candidates: URL[] = [];
  const seen = new Set<string>();

  for (const candidate of resources.icons.slice(0, browserIconMaxPageCandidates)) {
    appendCandidate(candidates, seen, candidate);
  }

  let manifestCandidates = 0;
  for (const manifestURL of resources.manifests.slice(0, browserIconMaxManifests)) {
    if (Date.now() >= deadline || manifestCandidates >= browserIconMaxManifestCandidates) {
      break;
    }
    for (const candidate of await loadManifestIconCandidates(manifestURL, deadline)) {
      const before = candidates.length;
      appendCandidate(candidates, seen, candidate);
      if (candidates.length > before) {
        manifestCandidates++;
      }
      if (manifestCandidates >= browserIconMaxManifestCandidates) {
        break;
      }
    }
  }

  for (const path of ['/favicon.ico', '/favicon.png', '/apple-touch-icon.png', '/apple-touch-icon-precomposed.png']) {
    appendCandidate(candidates, seen, new URL(path, resources.baseURL));
  }

  let lastError: unknown = new Error('browser icon fetch failed');
  for (const candidate of candidates.slice(0, browserIconMaxCandidates)) {
    if (Date.now() >= deadline) {
      break;
    }
    try {
      const blob = await fetchWithDeadline(
        candidate,
        {
          credentials: 'omit',
          headers: { Accept: 'image/avif,image/webp,image/png,image/jpeg,image/gif,image/x-icon,*/*;q=0.8' },
          mode: 'cors',
          redirect: 'follow'
        },
        deadline,
        async (response) => {
          if (!response.ok) {
            await cancelResponseBody(response);
            return null;
          }
          return readLimitedBlob(response, browserIconFileReadLimit);
        }
      );
      if (!blob || blob.size === 0) {
        continue;
      }
      return await uploadIcon(new File([blob], iconFileName(candidate), { type: blob.type }));
    } catch (error) {
      if (isUnauthorizedError(error)) {
        throw error;
      }
      lastError = error;
    }
  }
  throw lastError;
}

async function loadPageIconResources(baseURL: URL, deadline: number): Promise<BrowserIconResources> {
  try {
    const result = await fetchWithDeadline(
      baseURL,
      {
        credentials: 'omit',
        headers: { Accept: 'text/html,application/xhtml+xml' },
        mode: 'cors',
        redirect: 'follow'
      },
      deadline,
      async (response) => {
        if (!response.ok) {
          await cancelResponseBody(response);
          return null;
        }
        return {
          url: response.url,
          html: await readLimitedText(response, browserIconHTMLReadLimit)
        };
      }
    );
    if (!result) {
      return { icons: [], manifests: [], baseURL };
    }

    const finalURL = normalizeFetchURL(result.url) ?? baseURL;
    const document = new DOMParser().parseFromString(result.html, 'text/html');
    const icons: URL[] = [];
    const manifests: URL[] = [];

    for (const link of document.querySelectorAll('link[href]')) {
      const rel = link.getAttribute('rel') ?? '';
      const href = link.getAttribute('href') ?? '';
      const candidate = resolveFetchURL(href, finalURL);
      if (!candidate) {
        continue;
      }
      if (relIncludesIcon(rel)) {
        icons.push(candidate);
      } else if (relIncludesToken(rel, 'manifest')) {
        manifests.push(candidate);
      }
    }
    return { icons, manifests, baseURL: finalURL };
  } catch {
    return { icons: [], manifests: [], baseURL };
  }
}

async function loadManifestIconCandidates(manifestURL: URL, deadline: number): Promise<URL[]> {
  try {
    const result = await fetchWithDeadline(
      manifestURL,
      {
        credentials: 'omit',
        headers: { Accept: 'application/manifest+json,application/json' },
        mode: 'cors',
        redirect: 'follow'
      },
      deadline,
      async (response) => {
        if (!response.ok) {
          await cancelResponseBody(response);
          return null;
        }
        return {
          url: response.url,
          content: await readLimitedText(response, browserIconManifestReadLimit)
        };
      }
    );
    if (!result) {
      return [];
    }

    const manifest = JSON.parse(result.content) as WebManifest;
    const finalURL = normalizeFetchURL(result.url) ?? manifestURL;
    if (!Array.isArray(manifest.icons)) {
      return [];
    }
    return manifest.icons
      .map((icon) => {
        if (!icon || typeof icon !== 'object' || !('src' in icon) || typeof icon.src !== 'string') {
          return null;
        }
        return resolveFetchURL(icon.src, finalURL);
      })
      .filter((candidate): candidate is URL => candidate !== null);
  } catch {
    return [];
  }
}

async function fetchWithDeadline<T>(
  input: URL,
  init: RequestInit,
  deadline: number,
  consume: (response: Response) => Promise<T>
): Promise<T> {
  const remaining = deadline - Date.now();
  if (remaining <= 0) {
    throw new Error('browser icon fetch timed out');
  }

  const controller = new AbortController();
  const timer = window.setTimeout(() => controller.abort(), Math.min(remaining, browserIconRequestTimeoutMS));
  try {
    // 超时覆盖响应头和响应体读取，避免慢速内网站点长期占用管理端请求。
    const response = await fetch(input, { ...init, signal: controller.signal });
    return await consume(response);
  } finally {
    window.clearTimeout(timer);
  }
}

async function readLimitedText(response: Response, maxBytes: number): Promise<string> {
  return new TextDecoder().decode(await readLimitedBytes(response, maxBytes));
}

async function readLimitedBlob(response: Response, maxBytes: number): Promise<Blob> {
  const contentType = response.headers.get('Content-Type')?.split(';', 1)[0].trim() ?? '';
  const content = await readLimitedBytes(response, maxBytes);
  const buffer = new ArrayBuffer(content.byteLength);
  new Uint8Array(buffer).set(content);
  return new Blob([buffer], { type: contentType });
}

async function readLimitedBytes(response: Response, maxBytes: number): Promise<Uint8Array> {
  const declaredSize = Number(response.headers.get('Content-Length'));
  if (Number.isFinite(declaredSize) && declaredSize > maxBytes) {
    await cancelResponseBody(response);
    throw new Error('response is too large');
  }

  if (!response.body) {
    const content = await response.arrayBuffer();
    if (content.byteLength > maxBytes) {
      throw new Error('response is too large');
    }
    return new Uint8Array(content);
  }

  const reader = response.body.getReader();
  const chunks: Uint8Array[] = [];
  let total = 0;
  try {
    for (;;) {
      const { done, value } = await reader.read();
      if (done) {
        break;
      }
      total += value.byteLength;
      if (total > maxBytes) {
        await reader.cancel();
        throw new Error('response is too large');
      }
      chunks.push(value);
    }
  } finally {
    reader.releaseLock();
  }

  const content = new Uint8Array(total);
  let offset = 0;
  for (const chunk of chunks) {
    content.set(chunk, offset);
    offset += chunk.byteLength;
  }
  return content;
}

async function cancelResponseBody(response: Response) {
  if (!response.body) {
    return;
  }
  try {
    await response.body.cancel();
  } catch {
    // 响应可能已被浏览器关闭，无需影响后续候选尝试。
  }
}

function normalizeWebsiteURL(value: string): URL {
  const raw = value.trim();
  const parsed = normalizeFetchURL(raw.includes('://') ? raw : `https://${raw}`);
  if (!parsed) {
    throw new Error('invalid website URL');
  }
  return parsed;
}

function resolveFetchURL(value: string, baseURL: URL): URL | null {
  const raw = value.trim();
  if (!raw) {
    return null;
  }
  try {
    return normalizeFetchURL(new URL(raw, baseURL).toString());
  } catch {
    return null;
  }
}

function normalizeFetchURL(value: string): URL | null {
  try {
    const parsed = new URL(value);
    if ((parsed.protocol !== 'http:' && parsed.protocol !== 'https:') || parsed.username || parsed.password) {
      return null;
    }
    return parsed;
  } catch {
    return null;
  }
}

function appendCandidate(candidates: URL[], seen: Set<string>, candidate: URL) {
  if (seen.has(candidate.href) || candidates.length >= browserIconMaxCandidates) {
    return;
  }
  seen.add(candidate.href);
  candidates.push(candidate);
}

function relIncludesIcon(rel: string): boolean {
  return rel
    .toLocaleLowerCase()
    .split(/\s+/)
    .some((token) => token === 'icon' || token.endsWith('-icon'));
}

function relIncludesToken(rel: string, target: string): boolean {
  return rel.toLocaleLowerCase().split(/\s+/).includes(target);
}

function iconFileName(candidate: URL): string {
  const fileName = candidate.pathname.split('/').filter(Boolean).pop();
  return fileName || 'favicon';
}
