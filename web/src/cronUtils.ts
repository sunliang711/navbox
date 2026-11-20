import type { LanguageMode } from './preferences';

const descriptorsZh: Record<string, string> = {
  '@yearly': '每年 1 月 1 日 00:00',
  '@annually': '每年 1 月 1 日 00:00',
  '@monthly': '每月 1 日 00:00',
  '@weekly': '每周日 00:00',
  '@daily': '每天 00:00',
  '@midnight': '每天 00:00',
  '@hourly': '每小时整点'
};

const descriptorsEn: Record<string, string> = {
  '@yearly': 'Yearly on Jan 1 at 00:00',
  '@annually': 'Yearly on Jan 1 at 00:00',
  '@monthly': 'Monthly on day 1 at 00:00',
  '@weekly': 'Weekly on Sunday at 00:00',
  '@daily': 'Daily at 00:00',
  '@midnight': 'Daily at 00:00',
  '@hourly': 'Every hour on the hour'
};

const weekdaysZh = ['日', '一', '二', '三', '四', '五', '六', '日'];
const weekdaysEn = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];

const isNum = (value: string) => /^\d+$/.test(value);
const pad = (value: string) => value.padStart(2, '0');

function formatList(value: string, format: (item: string) => string, rangeSep: string, itemSep: string): string | null {
  const parts = value.split(',');
  const out: string[] = [];
  for (const part of parts) {
    const range = /^(\d+)-(\d+)$/.exec(part);
    if (range) {
      out.push(`${format(range[1])}${rangeSep}${format(range[2])}`);
    } else if (isNum(part)) {
      out.push(format(part));
    } else {
      return null;
    }
  }
  return out.join(itemSep);
}

export function describeCron(expr: string, lang: LanguageMode): string | null {
  const text = expr.trim();
  const isEn = lang === 'en';

  if (text.startsWith('@')) {
    const every = /^@every\s+(.+)$/.exec(text);
    if (every) {
      return isEn ? `Every ${every[1]}` : `每隔 ${every[1]}`;
    }
    return (isEn ? descriptorsEn[text] : descriptorsZh[text]) ?? null;
  }

  const fields = text.split(/\s+/);
  if (fields.length !== 5) {
    return null;
  }
  const [minute, hour, dom, month, dow] = fields;
  if (month !== '*') {
    return null;
  }

  if (hour === '*' && dom === '*' && dow === '*') {
    if (minute === '*') {
      return isEn ? 'Every minute' : '每分钟';
    }
    const step = /^\*\/(\d+)$/.exec(minute);
    if (step) {
      return isEn ? `Every ${step[1]} min` : `每 ${step[1]} 分钟`;
    }
    if (isNum(minute)) {
      if (isEn) {
        return minute === '0' ? 'Every hour' : `Hourly at :${pad(minute)}`;
      }
      return minute === '0' ? '每小时整点' : `每小时第 ${minute} 分`;
    }
    return null;
  }
  if (!isNum(minute)) {
    return null;
  }

  const hourStep = /^\*\/(\d+)$/.exec(hour);
  if (hourStep && dom === '*' && dow === '*') {
    if (isEn) {
      return minute === '0' ? `Every ${hourStep[1]} hours` : `Every ${hourStep[1]} hours at :${pad(minute)}`;
    }
    return minute === '0' ? `每 ${hourStep[1]} 小时整点` : `每 ${hourStep[1]} 小时，第 ${minute} 分`;
  }

  const rangeSep = isEn ? '–' : '至';
  const itemSep = isEn ? ', ' : '、';
  const times = formatList(hour, (h) => `${pad(h)}:${pad(minute)}`, rangeSep, itemSep);
  if (!times) {
    return null;
  }

  if (dom === '*' && dow === '*') {
    return isEn ? `Daily at ${times}` : `每天 ${times}`;
  }
  if (dom === '*') {
    const days = formatList(
      dow,
      (d) => (isEn ? weekdaysEn[Number(d)] ?? d : `周${weekdaysZh[Number(d)] ?? d}`),
      rangeSep,
      itemSep
    );
    if (!days) {
      return null;
    }
    return isEn ? `Every ${days} at ${times}` : `每${days} ${times}`;
  }
  if (dow === '*') {
    const days = formatList(dom, (d) => (isEn ? d : `${d} 日`), rangeSep, itemSep);
    if (!days) {
      return null;
    }
    return isEn ? `Monthly on day ${days} at ${times}` : `每月 ${days} ${times}`;
  }
  return null;
}

export function formatDateTime(iso?: string, lang: LanguageMode = 'zh'): string {
  if (!iso) {
    return '—';
  }
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) {
    return iso;
  }
  return d.toLocaleString(lang === 'en' ? 'en-US' : 'zh-CN', { hour12: false });
}

export function formatDuration(startedAt: string, finishedAt?: string): string {
  if (!finishedAt) {
    return '';
  }
  const ms = new Date(finishedAt).getTime() - new Date(startedAt).getTime();
  if (!Number.isFinite(ms) || ms < 0) {
    return '';
  }
  if (ms < 1000) {
    return `${ms}ms`;
  }
  if (ms < 60_000) {
    return `${(ms / 1000).toFixed(1)}s`;
  }
  const minutes = Math.floor(ms / 60_000);
  const seconds = Math.round((ms % 60_000) / 1000);
  return `${minutes}m${seconds}s`;
}

export async function copyToClipboard(text: string): Promise<void> {
  if (navigator.clipboard && window.isSecureContext) {
    await navigator.clipboard.writeText(text);
    return;
  }
  const textarea = document.createElement('textarea');
  textarea.value = text;
  textarea.style.position = 'fixed';
  textarea.style.opacity = '0';
  document.body.appendChild(textarea);
  textarea.select();
  const ok = document.execCommand('copy');
  document.body.removeChild(textarea);
  if (!ok) {
    throw new Error('copy failed');
  }
}
