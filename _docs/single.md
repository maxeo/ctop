# Single Container View

ctop provides a rolling, single container view for following metrics.
It fits the terminal size: the container details on top, the CPU, MEM, NET and IO
panels (side by side on wide terminals) and the environment variables, which scroll
and can be filtered with `/`. Enter selects a panel and the arrows move the
selection; Enter again opens its detail view (networks, ports, mounts, command,
limits, labels...). PgUp / PgDn show the previous or next container, left and right
(with nothing selected) or Esc go back to the list. See the README for every key.
<p align="center"><img width="80%" src="img/single.gif" alt="ctop"/></p>
