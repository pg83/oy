exec subreaper \
    /home/pg/monorepo/wirez/wirez -q \
    -F 127.0.0.1:1083 \
    -L 53:8.8.8.8:53/udp \
    -- /home/pg/monorepo/overseer/overseer "${@}"
