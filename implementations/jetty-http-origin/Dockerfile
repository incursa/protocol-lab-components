FROM jetty@sha256:542f79d7aaa27dd2c6f9092a36be4b5b443e9fd7efc7d43c673a959caafa0f53

USER root
RUN java -jar "$JETTY_HOME/start.jar" --add-modules=http,http2c,ee10-deploy

COPY source/OriginServlet.java /tmp/src/OriginServlet.java
COPY source/web.xml /tmp/web.xml
RUN mkdir -p /tmp/war/WEB-INF/classes \
    && cp /tmp/web.xml /tmp/war/WEB-INF/web.xml \
    && classpath="$(find "$JETTY_HOME/lib" -type f -name '*.jar' -print | paste -sd: -)" \
    && javac -encoding UTF-8 -cp "$classpath" -d /tmp/war/WEB-INF/classes /tmp/src/OriginServlet.java \
    && cd /tmp/war \
    && jar --create --file "$JETTY_BASE/webapps/root.war" . \
    && chown jetty:jetty "$JETTY_BASE/webapps/root.war" \
    && rm -rf /tmp/src /tmp/web.xml /tmp/war

USER jetty
ENV PLAB_HTTP_PORT=8080
EXPOSE 8080 8082
CMD ["sh", "-c", "exec java -jar \"$JETTY_HOME/start.jar\" \"jetty.http.port=${PLAB_HTTP_PORT:-8080}\""]
