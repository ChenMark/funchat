// FunChat iOS - 好友列表页 (FRD-004)
// 基于 DES-003 设计稿 P1

import SwiftUI

struct ChatListView: View {
    @State private var friends: [FriendItem] = []
    @State private var isLoading = false
    @State private var showAddFriend = false

    var body: some View {
        NavigationStack {
            Group {
                if friends.isEmpty && !isLoading {
                    // 空状态
                    EmptyStateView {
                        showAddFriend = true
                    }
                } else {
                    // 好友列表
                    List {
                        ForEach(friends) { friend in
                            NavigationLink(destination: ChatDetailView(friend: friend)) {
                                ChatListRow(friend: friend)
                            }
                        }
                        .onDelete(perform: deleteFriend)
                    }
                    .listStyle(.plain)
                    .refreshable {
                        await loadFriends()
                    }
                }
            }
            .navigationTitle("聊天")
            .navigationBarTitleDisplayMode(.large)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    NavigationLink(destination: AddFriendView()) {
                        Image(systemName: "plus")
                            .font(.system(size: 18))
                            .foregroundColor(Color("neutral-800"))
                    }
                }
            }
        }
        .task {
            await loadFriends()
        }
    }

    private func loadFriends() async {
        isLoading = true
        do {
            friends = try await APIClient.shared.getFriends()
        } catch {
            friends = []
        }
        isLoading = false
    }

    private func deleteFriend(at offsets: IndexSet) {
        guard let index = offsets.first else { return }
        let friend = friends[index]

        Task {
            do {
                _ = try await APIClient.shared.deleteFriend(friendId: friend.userId)
                await MainActor.run {
                    friends.remove(atOffsets: offsets)
                }
            } catch {
                // 错误处理
            }
        }
    }
}

// MARK: - 好友列表行

struct ChatListRow: View {
    let friend: FriendItem

    var body: some View {
        HStack(spacing: 12) {
            // 头像
            ZStack {
                Circle()
                    .fill(Color(hex: "FFD9D1"))
                    .frame(width: 36, height: 36)
                if friend.avatar.isEmpty {
                    Text(String(friend.nickname.prefix(1)))
                        .font(.system(size: 14, weight: .medium))
                        .foregroundColor(Color(hex: "FF6B4A"))
                } else {
                    // AsyncImage(url: URL(string: friend.avatar))
                }
            }

            // 昵称 + 最后消息
            VStack(alignment: .leading, spacing: 4) {
                Text(friend.nickname)
                    .font(.system(size: 16, weight: .semibold))
                    .foregroundColor(Color("neutral-800"))
                Text(friend.lastMsg)
                    .font(.system(size: 13))
                    .foregroundColor(Color("neutral-600"))
                    .lineLimit(1)
            }

            Spacer()

            // 时间 + 未读角标
            VStack(alignment: .trailing, spacing: 4) {
                Text(friend.lastTime)
                    .font(.system(size: 11))
                    .foregroundColor(Color("neutral-400"))
                if friend.unread > 0 {
                    Text("\(friend.unread)")
                        .font(.system(size: 11, weight: .bold))
                        .foregroundColor(.white)
                        .frame(minWidth: 18, minHeight: 18)
                        .background(Color.red)
                        .clipShape(Circle())
                }
            }
        }
        .padding(.vertical, 8)
    }
}

// MARK: - 空状态

struct EmptyStateView: View {
    let action: () -> Void

    var body: some View {
        VStack(spacing: 16) {
            Spacer()

            // 插画占位
            ZStack {
                Circle()
                    .fill(Color(hex: "FFF0ED"))
                    .frame(width: 200, height: 200)
                Text("💬")
                    .font(.system(size: 64))
            }

            VStack(spacing: 4) {
                Text("还没有聊天")
                    .font(.system(size: 18, weight: .semibold))
                    .foregroundColor(Color("neutral-800"))
                Text("去添加好友吧")
                    .font(.system(size: 15))
                    .foregroundColor(Color("neutral-600"))
            }

            Button(action: action) {
                Text("添加好友")
                    .font(.system(size: 16, weight: .semibold))
                    .foregroundColor(Color(hex: "FF6B4A"))
                    .frame(width: 140, height: 44)
                    .background(Color(hex: "FFD9D1"))
                    .cornerRadius(12)
            }
            .padding(.top, 8)

            Spacer()
        }
    }
}

// MARK: - 聊天详情页占位

struct ChatDetailView: View {
    let friend: FriendItem

    var body: some View {
        VStack {
            Text("与 \(friend.nickname) 的聊天")
                .font(.title)
            // TODO: Phase 2 实现完整聊天 UI
        }
        .navigationTitle(friend.nickname)
        .navigationBarTitleDisplayMode(.inline)
    }
}
