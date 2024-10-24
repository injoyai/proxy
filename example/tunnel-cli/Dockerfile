FROM alpine:3.8

LABEL maintainer="injoy"

###############################################################################
#                                INSTALLATION
###############################################################################

# 设置固定的项目路径
ENV WORKDIR /root/tunnel

# 添加应用可执行文件，并设置执行权限
ADD ./tunnel-cli   $WORKDIR/tunnel-cli
RUN chmod +x $WORKDIR/tunnel-cli

# 添加I18N多语言文件、静态文件、配置文件、模板文件
# ADD config   $WORKDIR/config

###############################################################################
#                                   START
###############################################################################
WORKDIR $WORKDIR
CMD $WORKDIR/tunnel-cli
