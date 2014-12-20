Getting Started
===============

You'll probably need to install a lot of dependencies. Good luck.

1. git (obviously)
2. lxc-docker (we've had success with 1.3.3)
3. fig (pip install?)


Initialize Database
===================

```
# Create data volume.
fig run psqldata

# Create tables.
fig run upgradedb
```


Compile Frontend
================

```
fig run frontend
```

You can also get live recompiling by keeping this running in the background:

```
fig run frontend gulp
```


Run Backend
===========

```
fit up backend
```
