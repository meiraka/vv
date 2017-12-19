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
      --mpd.addr string              mpd server address to connect (default "localhost:6600")
      --mpd.host string              [DEPRECATED] mpd server hostname to connect
      --mpd.music_directory string   set music_directory in mpd.conf value to search album cover image
      --mpd.network string           mpd server network to connect (default "tcp")
      --mpd.port string              [DEPRECATED] mpd server TCP port to connect
      --server.addr string           this app serving address (default ":8080")
      --server.keepalive             use HTTP keep-alive (default true)
      --server.port string           [DEPRECATED] this app serving TCP port


Configuration
=============

put `config.yaml <./appendix/example.config.yaml>`_ to /etc/xdg/vv/ or ~/.config/vv/
