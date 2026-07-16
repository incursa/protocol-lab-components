FROM nginx@sha256:d67ea0d64d518b1bb04acde3b00f722ac3e9764b3209a9b0a98924ba35e4b779

COPY nginx.conf.template /tmp/nginx.conf.template
RUN sed 's/${PLAB_HTTP_PORT}/8080/g' /tmp/nginx.conf.template > /etc/nginx/nginx.conf \
    && rm /tmp/nginx.conf.template

EXPOSE 8080
