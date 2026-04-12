PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin
COMPLETION_BASH ?= $(PREFIX)/share/bash-completion/completions
COMPLETION_ZSH  ?= $(PREFIX)/share/zsh/site-functions

.PHONY: install uninstall test lint

install:
	@mkdir -p $(DESTDIR)$(BINDIR)
	@cp bin/focus $(DESTDIR)$(BINDIR)/focus
	@chmod +x $(DESTDIR)$(BINDIR)/focus
	@echo "Installed focus to $(DESTDIR)$(BINDIR)/focus"
	@if [ -d "$(DESTDIR)$(COMPLETION_BASH)" ]; then \
		cp completions/focus.bash $(DESTDIR)$(COMPLETION_BASH)/focus; \
		echo "Installed bash completions"; \
	fi
	@if [ -d "$(DESTDIR)$(COMPLETION_ZSH)" ]; then \
		cp completions/_focus $(DESTDIR)$(COMPLETION_ZSH)/_focus; \
		echo "Installed zsh completions"; \
	fi

uninstall:
	@rm -f $(DESTDIR)$(BINDIR)/focus
	@rm -f $(DESTDIR)$(COMPLETION_BASH)/focus
	@rm -f $(DESTDIR)$(COMPLETION_ZSH)/_focus
	@echo "Uninstalled focus"

test:
	@bats test/

lint:
	@shellcheck bin/focus completions/focus.bash
