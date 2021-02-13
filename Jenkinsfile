pipeline {
    agent { 
	label 'master' 
    }
    tools {
	go 'Go 1.14'
    }
    environment {
	GO114MODULE = 'on'
        CGO_ENABLED = 0 
        GOOS = 'linux'
        GOARCH = 'amd64'
        GOPATH = "${JENKINS_HOME}/jobs/${JOB_NAME}/builds/${BUILD_ID}"
    }
    stages {
        stage('Cloning Git') {
            steps {
		sh 'rm -rf datera-csi'
		checkout([	$class: 'GitSCM', 
				branches: [[name: "${params.GIT_BRANCH}"]], 
				doGenerateSubmoduleConfigurations: false, 
				extensions: [[$class: 'RelativeTargetDirectory', relativeTargetDir: 'datera-csi'], 
					     [$class: 'CleanCheckout', deleteUntrackedNestedRepositories: true], 
					     [$class: 'CleanBeforeCheckout', deleteUntrackedNestedRepositories: true], 
					     [$class: 'WipeWorkspace']], 
				submoduleCfg: [], 
				userRemoteConfigs: [[credentialsId: '0e54e78d-bbf1-4a6c-840c-be582abefd62', 
						     url: 'https://github.com/Datera/datera-csi.git']]
			])
            }
        }
        stage('Build CSI driver image') {
            steps {
		// Run from under this directory only
                dir("datera-csi/cmd/dat-csi-plugin") {
               		sh "pwd"
                        script {
				// Set environment variables for Go build of CSI image
                                env.VERSION = sh(script:'cat ../../VERSION', returnStdout: true).trim()
				env.GITHASH = sh(script:'git describe --match nEvErMatch --always --abbrev=10', returnStdout: true).trim()
				env.NAME = 'dat-csi-plugin'
				env.GOSDK_V = sh(script:'go mod graph | grep "github.com/Datera/datera-csi github.com/Datera/go-sdk" | awk -F \"@\" "{print \\$2}"', returnStdout: true).trim()
				env.csi_driver_version_flag = "github.com/Datera/datera-csi/pkg/driver.Version=${env.VERSION}"
				env.gosdk_version_flag = "github.com/Datera/datera-csi/pkg/driver.SdkVersion=${env.GOSDK_V}"
				env.hash_flag = "github.com/Datera/datera-csi/pkg/driver.Githash=${env.GITHASH}"
				sh 'printenv'
				sh "go build -tags 'osusergo netgo static_build' -o ${env.NAME} -ldflags \"-X '${env.csi_driver_version_flag}' -X '${env.gosdk_version_flag}' -X '${env.hash_flag}'\" github.com/Datera/datera-csi/cmd/dat-csi-plugin"
				sh "ls -l dat-csi-plugin"
			}
		}
	    }
	}
        stage('Push CSI driver image to DockerHub') {
            steps {
		// Run from under this directory only
                dir("datera-csi/cmd/dat-csi-plugin") {
                        script {
				// The "dockerhub_creds" are Global Credentials created for this Pipeline 
                                docker.withRegistry('https://registry.hub.docker.com', "dockerhub_creds") {
                                        def csiDriverImage = docker.build("dateraiodev/dat-csi-plugin:${env.VERSION}", "-f Dockerfile ../..")
                                        sh "sudo docker images | grep ${env.VERSION}"
                                        csiDriverImage.push("${env.VERSION}")
                                        sh "sudo docker images --digests | grep ${env.VERSION}"
                                }
                        }
                }
            }
        }
        stage('Deploy to Dev cluster and run Regression') {
	    // Run this Deploy stage from the specified Agent node
	    // which has access to Datera cluster and regression scripts
            agent { 
		label 'csi_node' 
	    }
            steps {
		// Pass parameters to the next job Downstream
		build (job: "${params.REGRESSION_JOB}", parameters: [
        		[
            			$class: 'StringParameterValue',
            			name: 'CLUSTER',
            			value: "${params.CLUSTER}"
        		],
			[
                                $class: 'StringParameterValue',
                                name: 'CSI_DRIVER',
                                value: "${env.VERSION}"
                        ],
                	[
                        	$class: 'StringParameterValue',
                        	name: 'SETUP_CLUSTER',
                        	value: "${params.SETUP_CLUSTER}"
                	]
    		],
		wait: false)
	    }
	}
    }
}
