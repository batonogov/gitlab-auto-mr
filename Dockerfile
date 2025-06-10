FROM python:3.13-alpine

RUN pip install --no-cache \
    gitlab-auto-mr==1.2.0
