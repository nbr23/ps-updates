pipeline {
    agent any
    stages {
        stage('Checkout'){
            steps {
                checkout scm
            }
        }
        stage('Build') {
            steps {
                script {
                    env.REAL_PWD = getDockerPWD();
                    sh 'docker run --rm -w /app -v $REAL_PWD:/app golang:alpine go build'
                }
            }
        }
    }
}