export interface EnvEntry {
  key: string
  value: string
}

// Parse standard .env text into key/value entries.
// Blank lines and lines starting with '#' are ignored, an optional 'export ' prefix is stripped,
// and values wrapped in matching single/double quotes are unquoted.
export function parseEnvText(text: string): EnvEntry[] {
  const entries: EnvEntry[] = []
  const indexByKey = new Map<string, number>()

  for (const rawLine of text.split(/\r?\n/)) {
    const line = rawLine.trim()
    if (!line || line.startsWith('#')) continue

    const equalIndex = line.indexOf('=')
    if (equalIndex <= 0) continue

    const key = line.slice(0, equalIndex).replace(/^export\s+/, '').trim()
    if (!key) continue

    let value = line.slice(equalIndex + 1).trim()
    const quote = value[0]
    if (value.length >= 2 && quote === value[value.length - 1] && (quote === '"' || quote === "'")) {
      value = value.slice(1, -1)
      if (quote === '"') {
        value = value
          .replace(/\\n/g, '\n')
          .replace(/\\t/g, '\t')
          .replace(/\\"/g, '"')
          .replace(/\\\\/g, '\\')
      }
    }

    const existing = indexByKey.get(key)
    if (existing !== undefined) {
      entries[existing].value = value
    } else {
      indexByKey.set(key, entries.length)
      entries.push({ key, value })
    }
  }

  return entries
}

// Merge incoming entries into the current list.
// Existing keys are overwritten when `overwrite` is true, otherwise skipped; new keys are appended.
// Empty placeholder rows are dropped, and an empty list falls back to a single blank row.
export function mergeEnvEntries(current: EnvEntry[], incoming: EnvEntry[], overwrite = true): EnvEntry[] {
  const rows = current.filter((e) => e.key || e.value).map((e) => ({ key: e.key, value: e.value }))
  const indexByKey = new Map<string, number>()
  rows.forEach((e, i) => {
    if (e.key) indexByKey.set(e.key, i)
  })

  for (const item of incoming) {
    if (!item.key) continue
    const index = indexByKey.get(item.key)
    if (index !== undefined) {
      if (overwrite) rows[index].value = item.value
    } else {
      indexByKey.set(item.key, rows.length)
      rows.push({ key: item.key, value: item.value })
    }
  }

  return rows.length ? rows : [{ key: '', value: '' }]
}
