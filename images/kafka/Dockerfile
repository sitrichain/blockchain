# Copyright Greg Haskins All Rights Reserved
#
# SPDX-License-Identifier: Apache-2.0
#
FROM arm64v8/openjdk:8

ENV SCALA_VERSION=2.11 \
    KAFKA_VERSION=0.9.0.1 \
    KAFKA_DOWNLOAD_SHA1=FC9ED9B663DD608486A1E56197D318C41813D326

COPY kafka_"$SCALA_VERSION"-"$KAFKA_VERSION".tgz /opt/kafka_"$SCALA_VERSION"-"$KAFKA_VERSION".tgz

RUN tar xfz /opt/kafka_"$SCALA_VERSION"-"$KAFKA_VERSION".tgz -C /opt \
    && mv /opt/kafka_"$SCALA_VERSION"-"$KAFKA_VERSION" /opt/kafka \
    && rm /opt/kafka_"$SCALA_VERSION"-"$KAFKA_VERSION".tgz

ADD kafka-run-class.sh /opt/kafka/bin/kafka-run-class.sh

ADD docker-entrypoint.sh /docker-entrypoint.sh

EXPOSE 9092
EXPOSE 9093

ENTRYPOINT ["/docker-entrypoint.sh"]
CMD ["/opt/kafka/bin/kafka-server-start.sh"]
