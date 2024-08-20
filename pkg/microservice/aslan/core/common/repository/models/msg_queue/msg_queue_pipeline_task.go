/*
 * Copyright 2023 The KodeRover Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package msg_queue

import (
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/koderover/zadig/v2/pkg/microservice/aslan/core/common/repository/models/task"
)

type MsgQueuePipelineTask struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"             json:"id,omitempty"`
	Task      *task.Task
	QueueType string `json:"queue_type" bson:"queue_type"`
	QueueID   int    `json:"queue_id" bson:"queue_id"`
}

func (MsgQueuePipelineTask) TableName() string {
	return "msg_queue_pipeline_task"
}
