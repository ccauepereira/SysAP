#!/usr/bin/env python3
"""Experimental advisory reviewer for a local Git diff; it never blocks a PR."""

from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
QUESTIONS_PATH = Path(__file__).with_name("questions.json")


def git_output(*args: str) -> str:
    result = subprocess.run(
        ["git", *args],
        cwd=ROOT,
        check=False,
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        raise RuntimeError("não foi possível obter o diff Git")
    return result.stdout


def eligible_path(path: str) -> bool:
    parts = Path(path).parts
    return (
        not Path(path).is_absolute()
        and ".." not in parts
        and not any(part == ".env" or part.startswith(".env.") for part in parts)
        and not any(part in {".npmrc", ".pnpmrc", "id_rsa", "id_ed25519"} for part in parts)
        and not any(part.lower() in {"secrets", "credentials", ".git"} for part in parts)
        and Path(path).suffix.lower() not in {".pem", ".key", ".p12", ".pfx", ".jks", ".keystore"}
    )


def changed_paths(base: str | None) -> list[str]:
    args = ["diff", "--name-only"]
    if base:
        args.append(f"{base}...HEAD")
    else:
        args.append("HEAD")
    tracked = [path for path in git_output(*args).splitlines() if path and eligible_path(path)]
    if base:
        return tracked
    untracked = [
        path
        for path in git_output("ls-files", "--others", "--exclude-standard").splitlines()
        if path and eligible_path(path)
    ]
    return sorted(set(tracked + untracked))


def diff_text(base: str | None, relative_path: str) -> str:
    if not eligible_path(relative_path):
        return ""
    args = ["diff", "--no-ext-diff", "--no-textconv", "--unified=0"]
    if base:
        args.append(f"{base}...HEAD")
    else:
        args.append("HEAD")
    args.extend(["--", relative_path])
    tracked = git_output(*args)
    if base or tracked:
        return tracked
    candidate = ROOT / relative_path
    if candidate.is_symlink() or ROOT not in candidate.resolve().parents:
        return ""
    if not candidate.is_file() or candidate.stat().st_size > 1024 * 1024:
        return ""
    try:
        content = candidate.read_text(encoding="utf-8")
    except UnicodeDecodeError:
        return ""
    return "\n".join(f"+{line}" for line in content.splitlines())


def path_matches(path: str, prefixes: list[str]) -> bool:
    return any(path == prefix or path.startswith(prefix) for prefix in prefixes)


def main() -> int:
    parser = argparse.ArgumentParser(description="Laya PR gate experimental (advisory)")
    parser.add_argument("--base", help="commit/base ref para comparar com HEAD")
    args = parser.parse_args()

    try:
        rules = json.loads(QUESTIONS_PATH.read_text(encoding="utf-8"))
        if rules.get("mode") != "advisory" or not isinstance(rules.get("questions"), list):
            raise ValueError("questions.json inválido")
        paths = changed_paths(args.base)
        diffs = {path: diff_text(args.base, path) for path in paths}
    except (OSError, ValueError, RuntimeError):
        print("Laya advisory: indisponível; revisar manualmente.", file=sys.stderr)
        return 0

    print("Laya PR gate experimental — advisory; não bloqueia merge.")
    print(f"Arquivos analisados: {len(paths)}")
    if not paths:
        print("Sem alterações para revisar.")
        return 0

    applicable = [
        question
        for question in rules["questions"]
        if path_matches_any(paths, question.get("when_paths", []))
    ]
    print("Perguntas para revisão humana:")
    for question in applicable:
        print(f"- [{question.get('id', 'sem-id')}] {question.get('question', '')}")
    if not applicable:
        print("- Nenhuma pergunta contextual foi selecionada; revise escopo e impacto do diff.")

    signals = []
    for signal in rules.get("signals", []):
        pattern = signal.get("pattern")
        if (
            isinstance(pattern, str)
            and any(
                path_matches_any([path], signal.get("when_paths", []))
                and re.search(pattern, "\n".join(
                    line[1:] for line in diff.splitlines()
                    if line.startswith("+") and not line.startswith("+++")
                ))
                for path, diff in diffs.items()
            )
        ):
            signals.append(signal.get("message", "Sinal de revisão manual."))
    if signals:
        print("Sinais para investigação humana:")
        for signal in signals:
            print(f"- {signal}")
    else:
        print("Sinais automáticos: nenhum.")
    return 0


def path_matches_any(paths: list[str], prefixes: object) -> bool:
    return isinstance(prefixes, list) and any(path_matches(path, prefixes) for path in paths)


if __name__ == "__main__":
    sys.exit(main())
