FROM python:3.12.7-alpine

RUN pip install --no-cache \
        gitlab-auto-mr==1.2.0
