#@author Fred Brooker <git@gscloud.cz>

all:
	@echo "backup | fresh | clear | fix | exporter | sizer | build | db | stan";
	@echo "macro: everything";

fix:
	@php -f fixer.php

exporter:
	@php -f exporter.php
	@cp stats-* /home/mxdpeep/DS3/

sizer:
	@php -f sizer.php

clear:
	@-rm -f cache.json
	@-rm -f go/games.txt
	@echo "Done."

fresh:
	@-rm -f cache.json
	@-rm -f cache2.json
	@-rm -f go/games.*
	@echo "Done."

backup:
	@echo "Making backup ..."
	@rclone copy -P --exclude '.git/**' --exclude 'vendor/**' . gsc:metascore/

stan:
	@vendor/phpstan/phpstan/phpstan analyse -l 9 config.php  exporter.php  fixer.php  sizer.php

build:
	@echo "Building..."
	@cd go/ && rm -f metascore && go build -o metascore .

db:
	@cp games.txt go/
	@cd go/ && ./metascore
	
# macro
everything: build fix db sizer exporter
	@-git add -A
	@git commit -m "auto commit"
	@git push origin
	@echo "All done."
