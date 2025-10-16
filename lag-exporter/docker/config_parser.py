#!/usr/bin/env python3
import argparse, re, sys


def read_text(p):
    with open(p, "r", encoding="utf-8") as f:
        return f.read()


def extract_block(text, name):
    m = re.search(rf'(^|\s){re.escape(name)}\s*\{{', text, re.M)
    if not m: return ""
    i, depth, start = m.end(), 1, None
    while i < len(text):
        c = text[i]
        if c == '{':
            depth += 1
        elif c == '}':
            depth -= 1
            if depth == 0:
                return text[m.end():i]
        elif start is None:
            start = m.end()
        i += 1
    return ""


def strip_inline_comment(s):
    out, q = [], None
    i = 0
    while i < len(s):
        c = s[i]
        if q:
            if c == q: q = None
            out.append(c)
        else:
            if c in ('"', "'"):
                q = c;
                out.append(c)
            elif c == '#':
                break
            else:
                out.append(c)
        i += 1
    return "".join(out)


_rx_needs_q = re.compile(r'[^A-Za-z0-9._:/-]')


def needs_quotes(val: str) -> bool:
    return bool(_rx_needs_q.search(val))


def format_flag(flag_name: str, value) -> list[str]:
    out = []
    if value is None:
        out.append(flag_name)
    elif isinstance(value, list):
        for item in value:
            out.append(format_flag(flag_name, item)[0])
    else:
        v = str(value)
        out.append(f'{flag_name}="{v}"' if needs_quotes(v) else f"{flag_name}={v}")
    return out


def parse_block(block_text: str) -> list[tuple[str, object]]:
    lines = block_text.splitlines()
    i, res = 0, []
    while i < len(lines):
        raw = lines[i];
        i += 1
        line = strip_inline_comment(raw).strip()
        if not line: continue

        m_bare = re.match(r'^([A-Za-z0-9._-]+|--[A-Za-z0-9._-]+)$', line)
        if m_bare:
            key = m_bare.group(1).strip()
            res.append((key, None))
            continue

        m = re.match(r'^([A-Za-z0-9._-]+|--[A-Za-z0-9._-]+)\s*=\s*(.+)$', line)
        if not m:
            continue
        key, val = m.group(1).strip(), m.group(2).strip()

        if val.startswith('[') and not val.rstrip().endswith(']'):
            buf = [val]
            depth = val.count('[') - val.count(']')
            while i < len(lines) and depth > 0:
                nxt = strip_inline_comment(lines[i]);
                i += 1
                buf.append(nxt)
                depth += nxt.count('[') - nxt.count(']')
            val = "\n".join(buf)

        if val.startswith('[') and val.rstrip().endswith(']'):
            body = val.strip()[1:-1]
            items, parts = [], []
            for ln in body.splitlines():
                parts.extend(p.strip() for p in ln.split(','))
            for s in parts:
                if not s: continue
                s = s.strip()
                if (s.startswith('"') and s.endswith('"')) or (s.startswith("'") and s.endswith("'")):
                    s = s[1:-1]
                if s != "":
                    items.append(s)
            if items:
                res.append((key, items))
            continue

        if (val.startswith('"') and val.endswith('"')) or (val.startswith("'") and val.endswith("'")):
            val = val[1:-1]
        if val != "":
            res.append((key, val))
    return res


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--conf", default="/config/application.conf")
    ap.add_argument("--block", default="kafka_exporter")
    args = ap.parse_args()

    txt = read_text(args.conf)
    blk = extract_block(txt, args.block)
    if not blk:
        print(f'block "{args.block}" not found', file=sys.stderr)
        sys.exit(2)

    assignments = parse_block(blk)
    flags = []
    for key, val in assignments:
        flags.extend(format_flag(key, val))

    print(" ".join(flags))


if __name__ == "__main__":
    main()
