// //
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"github.com/gofrs/uuid"
	"github.com/aaron-skillz/sync-server-go/api"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *ApiServer) GetUsers(ctx context.Context, in *api.GetUsersRequest) (*api.Users, error) {
	// Before hook.
	if fn := s.runtime.BeforeGetUsers(); fn != nil {
		beforeFn := func(clientIP, clientPort string) error {
			result, err, code := fn(ctx, s.logger, ctx.Value(ctxUserIDKey{}).(uuid.UUID).String(), ctx.Value(ctxUsernameKey{}).(string), ctx.Value(ctxVarsKey{}).(map[string]string), ctx.Value(ctxExpiryKey{}).(int64), clientIP, clientPort, in)
			if err != nil {
				return status.Error(code, err.Error())
			}
			if result == nil {
				// If result is nil, requested resource is disabled.
				s.logger.Warn("Intercepted a disabled resource.", zap.Any("resource", ctx.Value(ctxFullMethodKey{}).(string)), zap.String("uid", ctx.Value(ctxUserIDKey{}).(uuid.UUID).String()))
				return status.Error(codes.NotFound, "Requested resource was not found.")
			}
			in = result
			return nil
		}

		// Execute the before function lambda wrapped in a trace for stats measurement.
		err := traceApiBefore(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), beforeFn)
		if err != nil {
			return nil, err
		}
	}

	if in.GetIds() == nil && in.GetUsernames() == nil && in.GetFacebookIds() == nil {
		return &api.Users{}, nil
	}

	ids := make([]string, 0)
	usernames := make([]string, 0)
	facebookIDs := make([]string, 0)

	if in.GetIds() != nil {
		for _, id := range in.GetIds() {
			if _, uuidErr := uuid.FromString(id); uuidErr != nil {
				return nil, status.Error(codes.InvalidArgument, "ID '"+id+"' is not a valid system ID.")
			}

			ids = append(ids, id)
		}
	}

	if in.GetUsernames() != nil {
		usernames = in.GetUsernames()
	}

	if in.GetFacebookIds() != nil {
		facebookIDs = in.GetFacebookIds()
	}

	users, err := GetUsers(ctx, s.logger, s.db, s.tracker, ids, usernames, facebookIDs)
	if err != nil {
		return nil, status.Error(codes.Internal, "Error retrieving user accounts.")
	}

	// After hook.
	if fn := s.runtime.AfterGetUsers(); fn != nil {
		afterFn := func(clientIP, clientPort string) error {
			return fn(ctx, s.logger, ctx.Value(ctxUserIDKey{}).(uuid.UUID).String(), ctx.Value(ctxUsernameKey{}).(string), ctx.Value(ctxVarsKey{}).(map[string]string), ctx.Value(ctxExpiryKey{}).(int64), clientIP, clientPort, users, in)
		}

		// Execute the after function lambda wrapped in a trace for stats measurement.
		traceApiAfter(ctx, s.logger, s.metrics, ctx.Value(ctxFullMethodKey{}).(string), afterFn)
	}

	return users, nil
}
