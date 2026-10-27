#!/bin/sh
# Opt-in: keep the local provider build current for Terraform dev_overrides.
#
# Installs git hooks that run `go install .` in the background after each
# commit, merge, branch checkout and rebase, in this clone and every worktree
# of it. The binary goes to $(go env GOBIN) or $(go env GOPATH)/bin, which is
# the directory your ~/.terraformrc dev_overrides should point at.
#
# Log: <git-common-dir>/local-install.log
# Remove: scripts/install-local-hooks.sh --uninstall
set -eu

hooks="$(git rev-parse --git-common-dir)/hooks"
marker="# quant-local-install"

if [ "${1:-}" = "--uninstall" ]; then
  for h in post-commit post-merge post-rewrite post-checkout; do
    if [ -f "$hooks/$h" ] && grep -q "$marker" "$hooks/$h"; then rm "$hooks/$h"; echo "removed $h"; fi
  done
  rm -f "$hooks/quant-local-install"
  exit 0
fi

cat > "$hooks/quant-local-install" <<'EOF'
#!/bin/sh
# quant-local-install
root=$(git rev-parse --show-toplevel) || exit 0
log="$(git rev-parse --git-common-dir)/local-install.log"
(
  cd "$root" || exit 0
  echo "$(date '+%F %T') $(git rev-parse --abbrev-ref HEAD)@$(git rev-parse --short HEAD) $root" >> "$log"
  if go install . >> "$log" 2>&1; then echo "  installed" >> "$log"; else echo "  FAILED" >> "$log"; fi
) </dev/null >/dev/null 2>&1 &
exit 0
EOF
chmod +x "$hooks/quant-local-install"

for h in post-commit post-merge post-rewrite post-checkout; do
  if [ -f "$hooks/$h" ] && ! grep -q "$marker" "$hooks/$h"; then
    echo "skipped $h: a hook already exists (add a call to $hooks/quant-local-install)"
    continue
  fi
  {
    echo '#!/bin/sh'
    echo "$marker"
    # post-checkout: $3 is 1 for a branch checkout, 0 for a file checkout
    [ "$h" = post-checkout ] && echo '[ "$3" = 1 ] || exit 0'
    echo 'exec "$(git rev-parse --git-common-dir)/hooks/quant-local-install"'
  } > "$hooks/$h"
  chmod +x "$hooks/$h"
  echo "installed $h"
done

go install . && echo "built $(go env GOPATH)/bin/terraform-provider-quant"
