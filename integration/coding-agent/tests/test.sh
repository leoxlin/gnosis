#!/bin/sh

if python3 -m unittest \
	&& python3 -c 'import calculator; assert calculator.add(2, 3) == 5' \
	&& grep -qx 'status: done' docs/directives/add-numbers.md \
	&& [ "$(grep -Fxc 'get procedures --tags gnosis,development' .gnosis-calls)" -eq 1 ] \
	&& grep -Fxq 'get procedures gnosis://core/procedures/development/implementing-directive.md --full' .gnosis-calls
then
	printf '1\n' >/logs/verifier/reward.txt
	exit 0
fi

printf '0\n' >/logs/verifier/reward.txt
exit 1
