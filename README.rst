==
vv
==

Web App client for Music Player Daemon

.. image:: appendix/screenshot.jpg
   :alt: screenshot


Installation
============

.. code-block:: shell

  go get github.com/meiraka/vv

Or get pre-built binary from `GitHub Releases page <https://github.com/meiraka/vv/releases>`_ and extract to somewhere you want.

Options
=======

.. code-block:: shell

  -d, --debug                        use local assets if exists
      --mpd.host string              mpd server hostname to connect (default "localhost")
      --mpd.music_directory string   set music_directory in mpd.conf value to search album cover image
      --mpd.port string              mpd server TCP port to connect (default "6600")
      --server.port string           this app serving TCP port (default "8080")

Configuration
=============

put `config.yaml <./appendix/example.config.yaml>`_ to /etc/xdg/vv/ or ~/.config/vv/
